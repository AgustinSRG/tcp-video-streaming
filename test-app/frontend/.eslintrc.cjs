module.exports = {
    root: true,
    env: {
        node: true,
        es2022: true,
    },
    extends: ["plugin:vue/vue3-essential", "eslint:recommended", "@vue/typescript/recommended"],
    parserOptions: {
        ecmaVersion: 2020,
    },
    rules: {
        "no-console": "off",
        "no-debugger": "warn",
        "no-useless-escape": "off",
        "@typescript-eslint/no-unsafe-declaration-merging": "off",
        "@typescript-eslint/no-explicit-any": "off",
        "@typescript-eslint/no-empty-function": "off",
        "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
        indent: "off",
        "vue/max-len": "off",
    },
    overrides: [
        {
            files: ["**/__tests__/*.{j,t}s?(x)", "**/tests/unit/**/*.spec.{j,t}s?(x)"],
            env: {
                mocha: true,
            },
        },
    ],
};
