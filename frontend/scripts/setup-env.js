// scripts/setup-env.js
import fs from 'fs';
import path from 'path';

export const setupEnv = () => {
  // Keep frontend and backend on the same explicitly selected environment.
  const configuredConfigPath = process.env.GENIX_CONFIG_FILE;
  const configPath = configuredConfigPath
    ? path.resolve(configuredConfigPath)
    : path.resolve(process.cwd(), '..', 'config.toml');
  console.log('📝 Generating .env files from ' + configPath + '...');
  try {
    if (fs.existsSync(configPath)) {
      const config = Bun.TOML.parse(fs.readFileSync(configPath, 'utf-8'));
      // Each selector option owns its API, bridge, and CDN configuration.
      const serializedPublicEndpoints = JSON.stringify(
        Array.isArray(config.endpoints) ? config.endpoints : []
      );
      const envContent = [
        `VITE_PROXY_PORT=${process.env.GENIX_PROXY_PORT || '3572'}`,
        `PUBLIC_ZONE_NAME=${config.frontend?.zone_name || ''}`,
        // Mirror the backend's configured standalone port for the local endpoint selector.
        `PUBLIC_LOCAL_API_PORT=${config.server?.port || 3589}`,
        `PUBLIC_ENDPOINTS=${serializedPublicEndpoints}`
      ].join('\n') + '\n';

      console.log("Seteando Enviroment:")
      console.log(envContent)

      fs.writeFileSync('.env', envContent);
      fs.writeFileSync(path.join('webpage', '.env'), envContent);
      console.log('✅ .env files created successfully');
      return true;
    } else {
      console.warn('⚠️  Config file not found at ' + configPath + ', skipping .env generation');
      return false;
    }
  } catch (error) {
    console.error('❌ Error generating .env files:', error);
    return false;
  }
};
