import fs from 'fs';
import path from 'path';

// Define input and output file names
const inputCssFile = 'fontello.css';
const inputWoff2File = 'fontello-prerender.woff2';
const outputCssFile = '../src/lib/fontello-prerender.css'
const id = 1

const embedWoff2FontBase64 = async () => {
    try {
        // 1. Read the content of fontello.css
        const cssContent = await fs.promises.readFile(inputCssFile, 'utf8');
        console.log(`Successfully read ${inputCssFile}`);

        // 2. Read the content of the WOFF2 font file
        // Note: WOFF2 is a binary format, so we don't specify 'utf8' encoding.
        const woff2Content = await fs.promises.readFile(inputWoff2File);
        console.log(`Successfully read ${inputWoff2File}`);

        // 3. Base64 Encode the WOFF2 content
        // The mime type for WOFF2 is 'font/woff2'
        const base64Woff2 = woff2Content.toString('base64');
        const woff2DataUri = `data:font/woff2;base64,${base64Woff2}`;
        console.log('WOFF2 content Base64 encoded.');

        // 4. Define the new @font-face and base icon styling template
        // This template includes the base64 WOFF2 font and the general icon styling.
        const fontFaceAndBaseIconTemplate = `
@font-face {
font-family: 'fontello${id}';
src: url('${woff2DataUri}') format('woff2');
font-weight: normal;
font-style: normal;
}

[class^="icon${id}-"]:before, [class*=" icon${id}-"]:before {
font-family: "fontello${id}";
font-style: normal;
font-weight: normal;
display: inline-block;
text-decoration: inherit;
width: 1em;
margin-right: .2em;
text-align: center;
font-variant: normal;
text-transform: none;
line-height: 1em;
margin-left: .2em;
}`;
        console.log('Defined new @font-face and base icon styling template.');

        // 5. Extract specific .icon-* class definitions from the original CSS
        // This regex captures individual icon rules like ".icon-camera:before { content: '\e800'; }"
        const specificIconRulesRegex = /(\.icon-[a-zA-Z0-9_-]+:before\s*\{[^}]+\})/g;
        let extractedIconRules = [];
        let match;
        // Use exec in a loop to get all matches
        while ((match = specificIconRulesRegex.exec(cssContent)) !== null) {
            match[1] = match[1].replace(`.icon-`,`.icon${id}-`)
            extractedIconRules.push(match[1]);
        }
        const iconRulesString = extractedIconRules.join('\n'); // Join extracted rules with newlines
        console.log(`Extracted ${extractedIconRules.length} specific icon rules.`);

        // 6. Assemble the final CSS content
        let finalCssContent = `/*\n * This CSS file contains the Fontello WOFF2 font embedded as a Base64 data URI.\n * It uses a custom @font-face template and extracted individual icon definitions.\n */\n\n`;
        finalCssContent += fontFaceAndBaseIconTemplate.trim() + '\n\n'; // Add template, trimmed, with extra newline for separation
        finalCssContent += iconRulesString.trim(); // Add extracted icon rules, trimmed

        // 7. Write the modified CSS content to the new file
        const outputDir = path.dirname(outputCssFile);
        await fs.promises.mkdir(outputDir, { recursive: true });

        await fs.promises.writeFile(outputCssFile, finalCssContent, 'utf8');
        console.log(`Successfully created ${outputCssFile} with embedded WOFF2 font and custom template.`);

    } catch (error) {
        console.error('An error occurred:', error.message);
        console.error(`Please ensure ${inputCssFile} and ${inputWoff2File} are in the same directory as the script.`);
    }
}

// Execute the function
embedWoff2FontBase64();
