#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-}"

if [[ -z "$ENV_FILE" ]]; then
  echo "usage: $0 <env-file>" >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing env file: $ENV_FILE" >&2
  exit 1
fi

perl - "$ENV_FILE" <<'PERL'
use strict;
use warnings;

my $env_file = shift @ARGV;
open my $fh, '<', $env_file or die "open $env_file: $!";

my %vars = %ENV;

sub trim {
  my ($value) = @_;
  $value =~ s/^\s+//;
  $value =~ s/\s+$//;
  return $value;
}

sub unquote {
  my ($value) = @_;
  if ($value =~ /\A"(.*)"\z/s || $value =~ /\A'(.*)'\z/s) {
    return $1;
  }
  return $value;
}

sub expand_vars {
  my ($value, $vars_ref) = @_;
  $value =~ s/\$\{([A-Z0-9_]+)\}/exists $vars_ref->{$1} ? $vars_ref->{$1} : (exists $ENV{$1} ? $ENV{$1} : "")/ge;
  return $value;
}

sub shell_quote {
  my ($value) = @_;
  $value =~ s/'/'"'"'/g;
  return "'$value'";
}

while (my $line = <$fh>) {
  chomp $line;
  $line =~ s/\r$//;
  next if $line =~ /^\s*$/;
  next if $line =~ /^\s*#/;
  next if $line =~ /^\s*export\s+/;

  my ($key, $raw_value) = $line =~ /^\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)\z/;
  die "invalid env line: $line\n" unless defined $key;

  my $value = trim($raw_value);
  $value = unquote($value);
  $value = expand_vars($value, \%vars);
  $vars{$key} = $value;

  print "export $key=", shell_quote($value), "\n";
}
PERL
