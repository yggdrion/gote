#!/usr/bin/env node

/**
 * Health check for vendor libraries
 */

const fs = require('fs');
const path = require('path');

const VENDOR_DIR = path.join(__dirname, '..', 'static', 'vendor');

function checkVendorHealth() {
    console.log('🔍 Checking vendor library health...');

    // Check if vendor directory exists
    if (!fs.existsSync(VENDOR_DIR)) {
        console.error('❌ Vendor directory does not exist:', VENDOR_DIR);
        process.exit(1);
    }

    // Required files
    const requiredFiles = [
        'marked.min.js',
        'marked-highlight.min.js',
        'highlight.min.js',
        'github.min.css',
        'versions.txt'
    ];

    const missingFiles = [];

    for (const file of requiredFiles) {
        const filePath = path.join(VENDOR_DIR, file);
        if (!fs.existsSync(filePath)) {
            missingFiles.push(file);
        }
    }

    if (missingFiles.length > 0) {
        console.error('❌ Missing required vendor files:');
        missingFiles.forEach(file => console.error(`   - ${file}`));
        console.error('');
        console.error('Run "npm run update-vendor" to download missing files.');
        process.exit(1);
    }

    console.log('✅ All required files present');

    // Validate JavaScript files
    console.log('🔍 Validating JavaScript files...');

    const jsFiles = ['marked.min.js', 'marked-highlight.min.js', 'highlight.min.js'];

    for (const jsFile of jsFiles) {
        try {
            const filePath = path.join(VENDOR_DIR, jsFile);
            const { Script } = require('vm');
            new Script(fs.readFileSync(filePath, 'utf8'));
            console.log(`✅ ${jsFile} is valid`);
        } catch (error) {
            console.error(`❌ ${jsFile} is not valid JavaScript:`, error.message);
            process.exit(1);
        }
    }

    // Check file sizes
    console.log('📊 File sizes:');

    const allFiles = ['marked.min.js', 'marked-highlight.min.js', 'highlight.min.js', 'github.min.css'];

    for (const file of allFiles) {
        const filePath = path.join(VENDOR_DIR, file);
        const stats = fs.statSync(filePath);
        console.log(`   ${file}: ${stats.size.toLocaleString()} bytes`);

        // Sanity checks for minimum file sizes
        if (file === 'marked.min.js' && stats.size < 10000) {
            console.warn(`⚠️  WARNING: ${file} seems unusually small (${stats.size} bytes)`);
        }
        if (file === 'marked-highlight.min.js' && stats.size < 1000) {
            console.warn(`⚠️  WARNING: ${file} seems unusually small (${stats.size} bytes)`);
        }
        if (file === 'highlight.min.js' && stats.size < 50000) {
            console.warn(`⚠️  WARNING: ${file} seems unusually small (${stats.size} bytes)`);
        }
        if (file === 'github.min.css' && stats.size < 500) {
            console.warn(`⚠️  WARNING: ${file} seems unusually small (${stats.size} bytes)`);
        }
    }

    // Show current versions
    const versionsPath = path.join(VENDOR_DIR, 'versions.txt');
    if (fs.existsSync(versionsPath)) {
        console.log('');
        console.log('📋 Current versions:');
        const versionsContent = fs.readFileSync(versionsPath, 'utf8');
        const versionLines = versionsContent.split('\n').filter(line =>
            line.includes('=') && !line.startsWith('#') && !line.includes('.url')
        );
        versionLines.forEach(line => console.log(`   ${line}`));
    }

    console.log('');
    console.log('✅ Vendor library health check passed!');
    console.log('🌐 Application can run completely offline');
}

// Run if called directly
if (require.main === module) {
    checkVendorHealth();
}

module.exports = { checkVendorHealth };
