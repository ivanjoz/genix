// Pure debug-logging helper: summarise a record for console output (numbers/strings
// kept as-is, arrays shown as `[len]`, nested objects as `{keys}`). Lives in its own
// module — NOT in service-worker-cache.ts — so app/SSR code (delta-cache.fetch.ts) can
// import it without dragging in that file's top-level `self.addEventListener` service
// worker side effects, which crash during prerender ("self is not defined").
export const parseObject = (rec: any) => {
	const newObject = {} as any
	for (const key in rec) {
		const values = rec[key]
		if (typeof values === 'number' || typeof values === 'string') {
			newObject[key] = values
		} else if (Array.isArray(values)) {
			newObject[key] = `[${values.length}]`
		} else if (values && typeof values === 'object') {
			newObject[key] = `{${Object.keys(values).join(", ")}}`
		}
	}
	return newObject
}
