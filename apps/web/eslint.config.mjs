import eslintPluginAstro from "eslint-plugin-astro";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    ignores: ["node_modules/**", "dist/**", ".astro/**"],
  },
  {
    files: ["**/*.{ts,tsx,mts,cts}"],
    extends: [...tseslint.configs.recommended],
  },
  ...eslintPluginAstro.configs["flat/recommended"],
  {
    files: ["**/*.{ts,tsx,mts,cts}"],
    rules: {
      "@typescript-eslint/no-unused-vars": "warn",
      "@typescript-eslint/no-explicit-any": "error",
    },
  },
);
