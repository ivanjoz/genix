#!/usr/bin/env bun
/**
 * Duplicate File Detection Script
 * 
 * This script scans the frontend folder and identifies duplicate files
 * based on content similarity > 95%.
 * 
 * REPORT-ONLY: Does not modify any files.
 * 
 * Usage: bun scripts/detect-duplicates.ts
 */

import { readdir, readFile } from 'fs/promises';
import { join, relative, extname, basename } from 'path';

// Configuration
const FRONTEND_DIR = process.cwd();
const SIMILARITY_THRESHOLD = 95;
const IGNORED_DIRS = [
  'node_modules',
  '.git',
  '.svelte-kit',
  'dist',
  'build',
  '.turbo',
  'coverage',
  '.next',
  'tmp'
];

// Extensions to check for duplicates
const CHECKED_EXTENSIONS = [
  '.ts', '.js', '.svelte', '.json', '.css', '.html',
  '.tsx', '.jsx'
];

interface FileComparison {
  file1: string;
  file2: string;
  similarity: number;
  type: 'exact' | 'similar';
}

interface FileContent {
  path: string;
  content: string;
}

/**
 * Check if a directory should be ignored
 */
function shouldIgnoreDir(dirPath: string): boolean {
  const parts = dirPath.split('/');
  return parts.some(part => IGNORED_DIRS.includes(part));
}

/**
 * Check if a file should be checked for duplicates
 */
function shouldCheckFile(filePath: string): boolean {
  const ext = extname(filePath).toLowerCase();
  return CHECKED_EXTENSIONS.includes(ext);
}

/**
 * Recursively get all files in a directory
 */
async function getAllFiles(dir: string): Promise<string[]> {
  const files: string[] = [];
  
  try {
    const entries = await readdir(dir, { withFileTypes: true });
    
    for (const entry of entries) {
      const fullPath = join(dir, entry.name);
      const relPath = relative(FRONTEND_DIR, fullPath);
      
      if (entry.isDirectory()) {
        if (!shouldIgnoreDir(relPath)) {
          const subFiles = await getAllFiles(fullPath);
          files.push(...subFiles);
        }
      } else if (entry.isFile() && shouldCheckFile(relPath)) {
        files.push(fullPath);
      }
    }
  } catch (error) {
    // Skip directories we can't read
  }
  
  return files;
}

/**
 * Calculate similarity between two strings using character comparison
 */
function calculateSimilarity(str1: string, str2: string): number {
  if (str1 === str2) return 100;
  
  // Normalize by removing extra whitespace (but preserve structure)
  const normalize = (s: string) => s.replace(/\s+/g, ' ').trim();
  const norm1 = normalize(str1);
  const norm2 = normalize(str2);
  
  if (norm1 === norm2) return 100;
  
  const len1 = norm1.length;
  const len2 = norm2.length;
  
  if (len1 === 0 && len2 === 0) return 100;
  if (len1 === 0 || len2 === 0) return 0;
  
  // Simple character-by-character comparison
  let matches = 0;
  const maxLen = Math.max(len1, len2);
  
  for (let i = 0; i < maxLen; i++) {
    const char1 = i < len1 ? norm1[i] : '';
    const char2 = i < len2 ? norm2[i] : '';
    
    if (char1 === char2) {
      matches++;
    }
  }
  
  return Math.round((matches / maxLen) * 100);
}

/**
 * Group files by their basename
 */
function groupFilesByBasename(files: string[]): Map<string, string[]> {
  const groups = new Map<string, string[]>();
  
  for (const file of files) {
    const base = basename(file);
    if (!groups.has(base)) {
      groups.set(base, []);
    }
    groups.get(base)!.push(file);
  }
  
  return groups;
}

/**
 * Compare files within each group
 */
async function compareFileGroups(groups: Map<string, string[]>): Promise<FileComparison[]> {
  const comparisons: FileComparison[] = [];
  
  for (const [basename, files] of groups.entries()) {
    if (files.length < 2) continue;
    
    // Load all file contents
    const fileContents: FileContent[] = [];
    for (const file of files) {
      try {
        const content = await readFile(file, 'utf-8');
        fileContents.push({ path: file, content });
      } catch (error) {
        console.warn(`Could not read file: ${file}`);
      }
    }
    
    // Compare all pairs
    for (let i = 0; i < fileContents.length; i++) {
      for (let j = i + 1; j < fileContents.length; j++) {
        const file1 = fileContents[i];
        const file2 = fileContents[j];
        
        const similarity = calculateSimilarity(file1.content, file2.content);
        
        if (similarity >= SIMILARITY_THRESHOLD) {
          comparisons.push({
            file1: relative(FRONTEND_DIR, file1.path),
            file2: relative(FRONTEND_DIR, file2.path),
            similarity,
            type: similarity === 100 ? 'exact' : 'similar'
          });
        }
      }
    }
  }
  
  return comparisons;
}

/**
 * Display results
 */
function displayResults(comparisons: FileComparison[]) {
  if (comparisons.length === 0) {
    console.log('\n✅ No duplicate files found with similarity >', SIMILARITY_THRESHOLD, '%');
    return;
  }
  
  // Sort by similarity (descending)
  comparisons.sort((a, b) => b.similarity - a.similarity);
  
  console.log('\n' + '='.repeat(80));
  console.log('DUPLICATE FILES DETECTION REPORT');
  console.log('='.repeat(80));
  console.log(`Threshold: ${SIMILARITY_THRESHOLD}% similarity`);
  console.log(`Total duplicates found: ${comparisons.length}\n`);
  
  // Group by type
  const exactMatches = comparisons.filter(c => c.type === 'exact');
  const similarMatches = comparisons.filter(c => c.type === 'similar');
  
  if (exactMatches.length > 0) {
    console.log('🔴 EXACT DUPLICATES (100% match):');
    console.log('-'.repeat(80));
    
    for (const comp of exactMatches) {
      console.log(`\n  File 1: ${comp.file1}`);
      console.log(`  File 2: ${comp.file2}`);
      console.log(`  Similarity: ${comp.similarity}%`);
    }
  }
  
  if (similarMatches.length > 0) {
    console.log('\n🟡 SIMILAR FILES (> 95% match):');
    console.log('-'.repeat(80));
    
    for (const comp of similarMatches) {
      console.log(`\n  File 1: ${comp.file1}`);
      console.log(`  File 2: ${comp.file2}`);
      console.log(`  Similarity: ${comp.similarity}%`);
    }
  }
  
  // Summary by package
  console.log('\n' + '='.repeat(80));
  console.log('SUMMARY BY PACKAGE:');
  console.log('='.repeat(80));
  
  const packageStats = new Map<string, number>();
  
  for (const comp of comparisons) {
    const pkg1 = comp.file1.split('/')[0];
    const pkg2 = comp.file2.split('/')[0];
    
    packageStats.set(pkg1, (packageStats.get(pkg1) || 0) + 1);
    packageStats.set(pkg2, (packageStats.get(pkg2) || 0) + 1);
  }
  
  for (const [pkg, count] of Array.from(packageStats.entries()).sort((a, b) => b[1] - a[1])) {
    console.log(`  ${pkg}: ${count} files involved in duplicates`);
  }
  
  console.log('\n' + '='.repeat(80));
  console.log('RECOMMENDATIONS:');
  console.log('='.repeat(80));
  console.log('1. Keep files in the lowest-level package according to DAG hierarchy');
  console.log('2. Remove copies from higher-level packages');
  console.log('3. Update imports to reference the canonical location');
  console.log('4. Consider using shared packages for truly shared functionality\n');
}

/**
 * Main execution
 */
async function main() {
  console.log('🔍 Scanning for duplicate files...');
  console.log(`   Directory: ${FRONTEND_DIR}`);
  console.log(`   Similarity threshold: ${SIMILARITY_THRESHOLD}%`);
  
  const startTime = Date.now();
  
  // Get all files
  console.log('\n📂 Collecting files...');
  const allFiles = await getAllFiles(FRONTEND_DIR);
  console.log(`   Found ${allFiles.length} files to check`);
  
  // Group by basename
  console.log('📊 Grouping files by name...');
  const fileGroups = groupFilesByBasename(allFiles);
  const groupsWithDuplicates = Array.from(fileGroups.entries()).filter(([_, files]) => files.length >= 2);
  console.log(`   Found ${groupsWithDuplicates.length} filenames with multiple occurrences`);
  
  // Compare files
  console.log('🔄 Comparing file contents...');
  const comparisons = await compareFileGroups(fileGroups);
  
  // Display results
  displayResults(comparisons);
  
  const duration = Date.now() - startTime;
  console.log(`\n✨ Analysis completed in ${duration}ms`);
}

// Run the script
main().catch(error => {
  console.error('❌ Error:', error);
  process.exit(1);
});