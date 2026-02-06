// scripts/setup-env.js
import fs from 'fs';
import path from 'path';

export const setupEnv = () => {
  console.log('📝 Generating .env files from credentials.json...');
  try {
    const credentialsPath = path.resolve(process.cwd(), '..', 'credentials.json');
    if (fs.existsSync(credentialsPath)) {
      const credentials = JSON.parse(fs.readFileSync(credentialsPath, 'utf-8'));
      const envContent = [
        `PUBLIC_LAMBDA_URL=${credentials.LAMBDA_URL || ''}`,
        `PUBLIC_SIGNALING_ENDPOINT=${credentials.SIGNALING_ENDPOINT || ''}`,
        `PUBLIC_SIGNALING_API_KEY=${credentials.SIGNALING_API_KEY || ''}`,
        `PUBLIC_FRONTEND_CDN=${credentials.FRONTEND_CDN || ''}`
      ].join('\n') + '\n';
      
      console.log("Seteando Enviroment:")
      console.log(envContent)

      fs.writeFileSync('.env', envContent);
      fs.writeFileSync(path.join('ecommerce', '.env'), envContent);
      console.log('✅ .env files created successfully');
      return true;
    } else {
      console.warn('⚠️  credentials.json not found at ' + credentialsPath + ', skipping .env generation');
      return false;
    }
  } catch (error) {
    console.error('❌ Error generating .env files:', error);
    return false;
  }
};
