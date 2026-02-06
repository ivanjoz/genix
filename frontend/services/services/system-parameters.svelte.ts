import { GetHandler, POST } from '$libs/http.svelte';

export interface ISystemParameter {
	ID: number;
	ValueText: string;
	ValueInts: number[];
	Value: number;
	Updated: number;
	UpdatedBy: number;
}

export const saveSystemParameters = (records: ISystemParameter[]) => {
	return POST({
		route: "system-parameters",
		data: records,
		refreshRoutes: ["system-parameters"]
	});
}

export class SystemParametersService extends GetHandler {
	route = "system-parameters"
	useCache = { min: 2, ver: 1 }

	records = $state([] as ISystemParameter[])
	recordsMap = $state(new Map<number, ISystemParameter>())

	handler(result: ISystemParameter[]): void {
		this.records = result;
		const newMap = new Map();
		for (const rec of result) {
			newMap.set(rec.ID, rec);
		}
		this.recordsMap = newMap;
		console.log("result getted:", result)
	}

	constructor() {
		super();
		this.fetch();
	}
}
