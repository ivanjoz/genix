import { join, resolve } from 'node:path'

interface FrontendEndpoint {
	name: string
	route: string
	bridge: string
	cdn_url: string
}

interface ProjectConfig {
	github_account?: unknown
	frontend?: { zone_name?: unknown }
	endpoints?: unknown
}

const repositoryRoot = join(import.meta.dir, '..')
// Follow the environment selected by deploy.sh while preserving standalone behavior.
const configuredConfigPath = process.env.GENIX_CONFIG_FILE?.trim()
const configPath = configuredConfigPath
	? resolve(configuredConfigPath)
	: join(repositoryRoot, 'config.toml')
const isDryRun = process.argv.includes('--dry-run')

const requireNonEmptyString = (configValue: unknown, fieldName: string): string => {
	if (typeof configValue !== 'string' || !configValue.trim()) {
		throw new Error(`Config field ${fieldName} must be a non-empty string`)
	}
	return configValue.trim()
}

const requireHttpUrl = (configValue: unknown, fieldName: string): string => {
	const configuredUrl = requireNonEmptyString(configValue, fieldName)
	const parsedUrl = new URL(configuredUrl)
	if (parsedUrl.protocol !== 'https:' && parsedUrl.protocol !== 'http:') {
		throw new Error(`Config field ${fieldName} must use HTTP or HTTPS`)
	}
	return configuredUrl
}

const requireFrontendEndpoints = (configValue: unknown): FrontendEndpoint[] => {
	if (!Array.isArray(configValue) || configValue.length === 0) {
		throw new Error('Config field endpoints must be a non-empty array')
	}

	return configValue.map((endpointValue, endpointIndex) => {
		if (!endpointValue || typeof endpointValue !== 'object') {
			throw new Error(`Config endpoints[${endpointIndex}] must be an object`)
		}
		const endpointRecord = endpointValue as Record<string, unknown>
		const endpointRoute = requireHttpUrl(endpointRecord.route, `endpoints[${endpointIndex}].route`)
		return {
			name: requireNonEmptyString(endpointRecord.name, `endpoints[${endpointIndex}].name`),
			route: endpointRoute,
			// A backend without an external bridge serves the agent stream itself.
			bridge: endpointRecord.bridge
				? requireHttpUrl(endpointRecord.bridge, `endpoints[${endpointIndex}].bridge`)
				: endpointRoute,
			cdn_url: requireHttpUrl(endpointRecord.cdn_url, `endpoints[${endpointIndex}].cdn_url`),
		}
	})
}

const runGitHubCommand = (
	argumentsList: string[],
	operationName: string,
	githubToken?: string,
): string => {
	let commandResult: ReturnType<typeof Bun.spawnSync>
	try {
		commandResult = Bun.spawnSync(['gh', ...argumentsList], {
			cwd: repositoryRoot,
			// GH_TOKEN selects the requested stored account without changing the
			// globally active gh login. Never place the token in command arguments.
			env: githubToken ? { ...process.env, GH_TOKEN: githubToken } : process.env,
			stdout: 'pipe',
			stderr: 'pipe',
		})
	} catch (commandError) {
		throw new Error(`Unable to execute gh while trying to ${operationName}: ${String(commandError)}`)
	}

	if (commandResult.exitCode !== 0) {
		const commandErrorOutput = commandResult.stderr.toString().trim()
		throw new Error(`Unable to ${operationName}${commandErrorOutput ? `: ${commandErrorOutput}` : ''}`)
	}
	return commandResult.stdout.toString().trim()
}

const main = async (): Promise<void> => {
	console.info(`[GitHub Vars] Reading public frontend configuration from ${configPath}`)
	if (!(await Bun.file(configPath).exists())) {
		throw new Error(`Config file was not found at ${configPath}`)
	}

	let config: ProjectConfig
	try {
		config = Bun.TOML.parse(await Bun.file(configPath).text()) as ProjectConfig
	} catch (parseError) {
		throw new Error(`Config file contains invalid TOML: ${String(parseError)}`)
	}

	const frontendEndpoints = requireFrontendEndpoints(config.endpoints)
	const githubAccount = requireNonEmptyString(config.github_account, 'github_account')
	const githubVariables = new Map<string, string>([
		['PUBLIC_ZONE_NAME', requireNonEmptyString(config.frontend?.zone_name, 'frontend.zone_name')],
		['PUBLIC_ENDPOINTS', JSON.stringify(frontendEndpoints)],
	])

	const githubToken = runGitHubCommand(
		['auth', 'token', '--hostname', 'github.com', '--user', githubAccount],
		`load the stored GitHub CLI token for ${githubAccount}`,
	)
	const authenticatedAccount = runGitHubCommand(
		['api', 'user', '--jq', '.login'],
		`verify the GitHub account ${githubAccount}`,
		githubToken,
	)
	if (authenticatedAccount.toLowerCase() !== githubAccount.toLowerCase()) {
		throw new Error(
			`github_account requested ${githubAccount}, but its stored token belongs to ${authenticatedAccount}`,
		)
	}

	const repositoryName = runGitHubCommand(
		['repo', 'view', '--json', 'nameWithOwner', '--jq', '.nameWithOwner'],
		'resolve the current GitHub repository',
		githubToken,
	)
	console.info(`[GitHub Vars] Authenticated account: ${authenticatedAccount}`)
	console.info(`[GitHub Vars] Target repository: ${repositoryName}`)
	console.info(`[GitHub Vars] Validated ${frontendEndpoints.length} API endpoint(s)`)

	for (const [variableName, variableValue] of githubVariables) {
		if (isDryRun) {
			console.info(`[GitHub Vars] Dry run: would set ${variableName}`)
			continue
		}
		runGitHubCommand(
			['variable', 'set', variableName, '--body', variableValue, '--repo', repositoryName],
			`set repository variable ${variableName}`,
			githubToken,
		)
		console.info(`[GitHub Vars] Set ${variableName}`)
	}

	console.info(isDryRun
		? '[GitHub Vars] Dry run completed; GitHub was not modified.'
		: '[GitHub Vars] Frontend repository variables configured successfully.')
}

await main().catch((scriptError) => {
	console.error(`[GitHub Vars] ${scriptError instanceof Error ? scriptError.message : String(scriptError)}`)
	process.exit(1)
})
