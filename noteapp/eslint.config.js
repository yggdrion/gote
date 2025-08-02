export default [
    {
        files: ['static/**/*.js'],
        ignores: ['static/vendor/**/*.js'],
        languageOptions: {
            ecmaVersion: 2021,
            sourceType: 'module',
            globals: {
                window: 'readonly',
                document: 'readonly',
                navigator: 'readonly',
                location: 'readonly',
                // add more browser globals as needed
            },
        },
        // Add rules and plugins as needed
    },
];
