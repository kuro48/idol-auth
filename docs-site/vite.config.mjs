import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

const root = fileURLToPath(new URL(".", import.meta.url));
const page = (path) => resolve(root, path, "index.html");

export default defineConfig({
  base: "./",
  build: {
    rollupOptions: {
      input: {
        main: page("."),
        start: page("start"),
        concepts: page("concepts"),
        sdk: page("sdk"),
        management: page("management"),
        account: page("account"),
        security: page("security"),
        api: page("api"),
      },
    },
  },
});
