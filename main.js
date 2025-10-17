// ABOUTME: Main entry point for the Uplift frontend
// ABOUTME: Initializes Alpine.js and imports the uplift app logic
import Alpine from 'alpinejs';
import './src/css/styles.css';
import './src/js/app.js';

// Make Alpine available globally
window.Alpine = Alpine;

// Start Alpine
Alpine.start();
