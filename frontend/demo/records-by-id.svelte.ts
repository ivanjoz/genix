import { getRecordByID } from "../libs/cache-by-ids.svelte"

export const testRecordsByIDs = () => {
	
	const ids = [1,4,5,6]
	const promises = []
	
	for (const id of ids) {
		const promise = getRecordByID("productos-ids", id)
		promises.push(promise)
	}
	
	console.log("productos... getting ids::", ids)
	
	Promise.all(promises).then(results => {
		console.log("productos getted by id::", results)
	})
}
