const fs = require('fs');
const path = require('path');
const { glob } = require('glob');
const coffee = require('coffeescript');
const ts = require('typescript');

// Function to parse step definitions from JavaScript/TypeScript files
function parseJsOrTsSteps(content, fileType = 'javascript') {
  const steps = [];
  // Match different step definition patterns
  const patterns = [
    // String patterns
    /(Given|When|Then|And)\s*\(\s*['"`](.*?)['"`]/g,
    // Regex patterns
    /(Given|When|Then|And)\s*\(\s*\/(.*?)\/[gim]*\s*[,)]/g,
    // Template literal patterns
    /(Given|When|Then|And)\s*\(\s*`(.*?)`/g,
    // TypeScript decorated patterns
    /@(Given|When|Then|And)\s*\(\s*['"`](.*?)['"`]\s*\)/g,
    // TypeScript decorator with regex
    /@(Given|When|Then|And)\s*\(\s*\/(.*?)\/[gim]*\s*\)/g
  ];

  patterns.forEach(pattern => {
    let match;
    while ((match = pattern.exec(content)) !== null) {
      const [_, type, stepPattern] = match;
      steps.push({
        type: type,
        pattern: stepPattern,
        fileType: fileType,
        patternType: pattern.toString().includes('/') ? 'regex' : 'string'
      });
    }
  });

  return steps;
}

// Function to parse TypeScript files
function parseTsSteps(content) {
  try {
    // First try to parse as regular TypeScript
    const steps = parseJsOrTsSteps(content, 'typescript');

    // If no steps found, try transpiling to JavaScript first
    if (steps.length === 0) {
      console.log('No steps found with direct TypeScript parsing, trying transpilation...');
      const transpiled = ts.transpileModule(content, {
        compilerOptions: {
          target: ts.ScriptTarget.ES2020,
          module: ts.ModuleKind.CommonJS,
          experimentalDecorators: true
        }
      });

      const transpiledSteps = parseJsOrTsSteps(transpiled.outputText, 'typescript');
      steps.push(...transpiledSteps);
    }

    return steps;
  } catch (error) {
    console.error('Error parsing TypeScript file:', error);
    return [];
  }
}

// Function to parse step definitions from CoffeeScript files
function parseCoffeeSteps(content) {
  const steps = [];
  // Match different CoffeeScript step definition patterns
  const patterns = [
    // @ syntax with strings
    /@(Given|When|Then|And)\s+['"`](.*?)['"`]/g,
    // @ syntax with regex
    /@(Given|When|Then|And)\s+\/(.*?)\/[gim]*\s*/g,
    // Without @ syntax, strings
    /(Given|When|Then|And)\s+['"`](.*?)['"`]/g,
    // Without @ syntax, regex
    /(Given|When|Then|And)\s+\/(.*?)\/[gim]*\s*/g
  ];

  patterns.forEach(pattern => {
    let match;
    while ((match = pattern.exec(content)) !== null) {
      const [_, type, stepPattern] = match;
      steps.push({
        type: type,
        pattern: stepPattern,
        fileType: 'coffeescript',
        patternType: pattern.toString().includes('/') ? 'regex' : 'string'
      });
    }
  });

  // If no steps found with direct patterns, try compiling to JS
  if (steps.length === 0) {
    try {
      console.log('No steps found with direct CoffeeScript parsing, trying JS compilation...');
      const jsContent = coffee.compile(content, { bare: true });
      const jsSteps = parseJsOrTsSteps(jsContent, 'coffeescript');
      steps.push(...jsSteps);
    } catch (error) {
      console.error('Error parsing CoffeeScript file:', error);
    }
  }

  return steps;
}

async function findStepFiles(startPath) {
  try {
    const absolutePath = path.resolve(startPath);
    if (!fs.existsSync(absolutePath)) {
      throw new Error(`Directory does not exist: ${absolutePath}`);
    }

    console.log(`Searching for step files in: ${absolutePath}`);

    const files = await glob('**/step_definitions/**/*.{js,coffee,ts}', {
      cwd: absolutePath,
      nocase: true,  // Make the search case-insensitive
      ignore: [
        '**/node_modules/**',
        '**/dist/**',
        '**/build/**',
        '**/coverage/**',
        '**/tmp/**',
        '**/temp/**'
      ]
    });

    // Log found files by extension
    const filesByExt = files.reduce((acc, file) => {
      const ext = path.extname(file);
      acc[ext] = (acc[ext] || 0) + 1;
      return acc;
    }, {});

    console.log('\nFiles found by extension:');
    Object.entries(filesByExt).forEach(([ext, count]) => {
      console.log(`${ext}: ${count} files`);
    });

    return files;
  } catch (err) {
    throw new Error(`Error finding step files: ${err.message}`);
  }
}

// Main function to process all files and generate JSON
async function generateStepsJson(searchPath) {
  try {
    const files = await findStepFiles(searchPath);
    const allSteps = [];

    for (const file of files) {
      const fullPath = path.join(searchPath, file);
      console.log(`Processing file: ${file}`);

      const content = fs.readFileSync(fullPath, 'utf8');
      const fileExt = path.extname(file).toLowerCase();

      let steps;
      if (fileExt === '.coffee') {
        console.log('Parsing as CoffeeScript:', file);
        steps = parseCoffeeSteps(content);
      } else if (fileExt === '.ts') {
        console.log('Parsing as TypeScript:', file);
        steps = parseTsSteps(content);
      } else {
        console.log('Parsing as JavaScript:', file);
        steps = parseJsOrTsSteps(content);
      }

      if (steps.length > 0) {
        console.log(`Found ${steps.length} steps in ${file}`);
        allSteps.push({
          file: file,
          steps: steps
        });
      } else {
        console.log(`No steps found in ${file}`);
      }
    }

    const output = {
      searchPath: searchPath,
      generatedAt: new Date().toISOString(),
      totalFiles: files.length,
      totalSteps: allSteps.reduce((acc, file) => acc + file.steps.length, 0),
      stepDefinitions: allSteps
    };

    // Write the result to a JSON file
    const outputFile = 'cucumber-steps.json';
    fs.writeFileSync(outputFile, JSON.stringify(output, null, 2));
    console.log(`\nSummary:`);
    console.log(`- Total files processed: ${output.totalFiles}`);
    console.log(`- Total steps found: ${output.totalSteps}`);
    console.log(`- Output written to: ${outputFile}`);
  } catch (error) {
    console.error('Error generating steps JSON:', error);
    process.exit(1);
  }
}

// Get the search path from command line arguments
const searchPath = process.argv[2] || process.cwd();

// Run the program
generateStepsJson(searchPath);
