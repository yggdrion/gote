#!/usr/bin/env node

/**
 * Update vendor files from node_modules
 * This script syncs the static/vendor/ directory with the versions in package.json
 */

const fs = require('fs');
const path = require('path');
const https = require('https');

const VENDOR_DIR = path.join(__dirname, '..', 'static', 'vendor');
const PACKAGE_JSON = path.join(__dirname, '..', 'package.json');

// CDN URLs for the libraries
const CDN_URLS = {
    'marked': 'https://cdn.jsdelivr.net/npm/marked@{version}/lib/marked.umd.js',
    'highlight.js': {
        js: 'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/highlight.min.js',
        css: 'https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/styles/github.min.css'
    }
};

// File mappings
const FILE_MAPPINGS = {
    'marked': 'marked.min.js',
    'highlight.js': {
        js: 'highlight.min.js',
        css: 'github.min.css'
    }
};

async function downloadFile(url, outputPath) {
    return new Promise((resolve, reject) => {
        console.log(`📥 Downloading ${url}...`);

        const file = fs.createWriteStream(outputPath);
        https.get(url, (response) => {
            if (response.statusCode !== 200) {
                reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
                return;
            }

            response.pipe(file);
            file.on('finish', () => {
                file.close();
                console.log(`✅ Saved to ${outputPath}`);
                resolve();
            });
        }).on('error', (err) => {
            fs.unlink(outputPath, () => { }); // Delete the file on error
            reject(err);
        });
    });
} async function validateJavaScript(filePath) {
    try {
        // Use Node.js to validate JavaScript syntax
        const { Script } = require('vm');
        new Script(fs.readFileSync(filePath, 'utf8'));
        console.log(`✅ ${path.basename(filePath)} is valid JavaScript`);
        return true;
    } catch (error) {
        console.error(`❌ ${path.basename(filePath)} is invalid: ${error.message}`);
        return false;
    }
}

async function updateVendorFiles() {
    try {
        // Read package.json
        const packageJson = JSON.parse(fs.readFileSync(PACKAGE_JSON, 'utf8'));
        const dependencies = packageJson.dependencies;

        console.log('🔄 Updating vendor files from package.json...');
        console.log(`📦 Dependencies:`, dependencies);
        console.log(`📁 Working directory: ${process.cwd()}`);
        console.log(`📄 Package.json path: ${PACKAGE_JSON}`);

        // Debug: Check if node_modules version matches package.json
        if (dependencies.marked) {
            const markedPkgPath = path.join(__dirname, '..', 'node_modules', 'marked', 'package.json');
            if (fs.existsSync(markedPkgPath)) {
                const markedPkg = JSON.parse(fs.readFileSync(markedPkgPath, 'utf8'));
                console.log(`🔍 node_modules marked version: ${markedPkg.version}`);
                console.log(`🔍 package.json marked spec: ${dependencies.marked}`);
            } else {
                console.log('⚠️  marked not found in node_modules');
            }
        }

        // Ensure vendor directory exists
        if (!fs.existsSync(VENDOR_DIR)) {
            fs.mkdirSync(VENDOR_DIR, { recursive: true });
        }

        // Create backups
        const backupDir = path.join(VENDOR_DIR, 'backup-' + Date.now());
        fs.mkdirSync(backupDir);

        const vendorFiles = fs.readdirSync(VENDOR_DIR).filter(f => f.endsWith('.js') || f.endsWith('.css'));
        for (const file of vendorFiles) {
            fs.copyFileSync(path.join(VENDOR_DIR, file), path.join(backupDir, file));
        }
        console.log(`💾 Created backup in ${backupDir}`);

        let allValid = true;

        // Update marked.js
        if (dependencies.marked) {
            const version = dependencies.marked.replace(/[\^~]/, '');
            const url = CDN_URLS.marked.replace('{version}', version);
            const outputPath = path.join(VENDOR_DIR, FILE_MAPPINGS.marked);

            await downloadFile(url, outputPath);
            if (!await validateJavaScript(outputPath)) {
                allValid = false;
            }
        }

        // Update highlight.js
        if (dependencies['highlight.js']) {
            const version = dependencies['highlight.js'].replace(/[\^~]/, '');

            // Download JS file
            const jsUrl = CDN_URLS['highlight.js'].js.replace('{version}', version);
            const jsPath = path.join(VENDOR_DIR, FILE_MAPPINGS['highlight.js'].js);
            await downloadFile(jsUrl, jsPath);
            if (!await validateJavaScript(jsPath)) {
                allValid = false;
            }

            // Download CSS file
            const cssUrl = CDN_URLS['highlight.js'].css.replace('{version}', version);
            const cssPath = path.join(VENDOR_DIR, FILE_MAPPINGS['highlight.js'].css);
            await downloadFile(cssUrl, cssPath);
        }

        if (!allValid) {
            console.error('❌ Some files failed validation. Restoring from backup...');

            // Restore from backup
            for (const file of vendorFiles) {
                fs.copyFileSync(path.join(backupDir, file), path.join(VENDOR_DIR, file));
            }

            process.exit(1);
        }

        // Update versions.txt
        const versionsPath = path.join(VENDOR_DIR, 'versions.txt');
        const markedVersion = dependencies.marked?.replace(/[\^~]/, '') || 'unknown';
        const highlightVersion = dependencies['highlight.js']?.replace(/[\^~]/, '') || 'unknown';

        const versionsContent = [
            '# Vendor Library Versions',
            '# This file tracks the versions of locally stored vendor libraries',
            '',
            `marked.js=${markedVersion}`,
            `highlight.js=${highlightVersion}`,
            '',
            '# Update URLs',
            `marked.js.url=https://cdn.jsdelivr.net/npm/marked@{version}/lib/marked.umd.js`,
            'highlight.js.url=https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/highlight.min.js',
            'highlight.js.css.url=https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/styles/github.min.css',
            '',
            `# Last updated`,
            `last_updated=${new Date().toISOString().split('T')[0]}`
        ].join('\n');

        fs.writeFileSync(versionsPath, versionsContent);
        console.log('📝 Updated versions.txt');

        // Clean up backup if everything succeeded
        fs.rmSync(backupDir, { recursive: true });
        console.log('🧹 Cleaned up backup');

        console.log('✅ Vendor files updated successfully!');

    } catch (error) {
        console.error('❌ Error updating vendor files:', error.message);
        process.exit(1);
    }
}

// Run if called directly
if (require.main === module) {
    updateVendorFiles();
}

module.exports = { updateVendorFiles };
