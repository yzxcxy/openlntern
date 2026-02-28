import js from "@eslint/js";
import eslintConfigPrettier from "eslint-config-prettier";
import tseslint from "typescript-eslint";
import pluginReactHooks from "eslint-plugin-react-hooks";
import pluginReact from "eslint-plugin-react";
import globals from "globals";
import pluginNext from "@next/eslint-plugin-next";
import turboPlugin from "eslint-plugin-turbo";
import onlyWarn from "eslint-plugin-only-warn";

const HARD_CODED_PALETTE_CLASS_PATTERN =
  "(?:^|\\s)(?:bg|text|border|from|via|to)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-\\d{2,3}(?=\\s|$)";
const HARD_CODED_ARBITRARY_COLOR_PATTERN =
  "(?:^|\\s)(?:bg|text|border)-\\[[^\\]]*(?:#|rgba?|hsla?|linear-gradient)[^\\]]*\\]";

/**
 * A shared ESLint configuration for the repository.
 *
 * @type {import("eslint").Linter.Config[]}
 * */
 const baseConfig = [
  js.configs.recommended,
  eslintConfigPrettier,
  ...tseslint.configs.recommended,
  {
    plugins: {
      turbo: turboPlugin,
    },
    rules: {
      "turbo/no-undeclared-env-vars": "warn",
    },
  },
  {
    plugins: {
      onlyWarn,
    },
  },
  {
    ignores: ["dist/**"],
  },
  {
    files: ["app/**/*.{ts,tsx}"],
    plugins: {
      react: pluginReact,
    },
    settings: { react: { version: "detect" } },
    languageOptions: {
      globals: {
        ...globals.serviceworker,
      },
    },
  },
  {
    plugins: {
      "@next/next": pluginNext,
    },
    rules: {
      ...pluginNext.configs.recommended.rules,
      ...pluginNext.configs["core-web-vitals"].rules,
    },
  },
  {
    plugins: {
      "react-hooks": pluginReactHooks,
    },
    rules: {
      ...pluginReactHooks.configs.recommended.rules,
    },
  },
  {
    files: ["app/**/*.{ts,tsx}"],
    ignores: ["app/components/ui/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-syntax": [
        "warn",
        {
          selector:
            "JSXOpeningElement[name.name='button'], JSXOpeningElement[name.name='input'], JSXOpeningElement[name.name='select'], JSXOpeningElement[name.name='textarea']",
          message:
            "Use app/components/ui primitives instead of raw form controls outside the shared UI layer.",
        },
        {
          selector: `JSXAttribute[name.name='className'] > Literal[value=/${HARD_CODED_PALETTE_CLASS_PATTERN}/]`,
          message:
            "Avoid hard-coded Tailwind palette classes in page code. Use semantic tokens or shared UI variants.",
        },
        {
          selector: `JSXAttribute[name.name='className'] > Literal[value=/${HARD_CODED_ARBITRARY_COLOR_PATTERN}/]`,
          message:
            "Avoid raw rgba/hex/gradient class values in page code. Move the style into semantic tokens or shared components.",
        },
      ],
    },
  },
];

export default baseConfig;
