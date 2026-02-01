import postcss from 'postcss';
import tailwindcss from '@tailwindcss/vite';
import fs from 'fs';
import path from 'path';

const css = fs.readFileSync('routes/app.css', 'utf8');

async function run() {
  try {
    console.log('Starting PostCSS processing...');
    // Note: @tailwindcss/vite is a Vite plugin, for PostCSS we usually use 'tailwindcss' package
    // But in v4, 'tailwindcss' IS the postcss plugin.
    const tailwind = (await import('tailwindcss')).default;
    
    const result = await postcss([
      tailwind()
    ]).process(css, { from: 'routes/app.css', to: 'repro_out.css' });
    
    fs.writeFileSync('repro_out.css', result.css);
    console.log('Successfully processed CSS. Output written to repro_out.css');
  } catch (error) {
    console.error('PostCSS Error:');
    console.error(error.message);
    if (error.line) {
      console.error(`Line: ${error.line}, Column: ${error.column}`);
      const lines = css.split('\n');
      console.error(`Context: ${lines[error.line - 1]}`);
    }
    // Also log the stack trace if needed
    // console.error(error.stack);
    
    // If it fails, let's try to see if we can get the expanded CSS anyway 
    // (though usually PostCSS stops on error)
  }
}

run();
