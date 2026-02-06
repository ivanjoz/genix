import { GetHandler, POST } from '$core/http.svelte';

export interface ISystemParameter {
	ParameterID: number;
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
	useCache = { min: 10, ver: 1 }

	records = $state([] as ISystemParameter[])
	recordsMap = $state(new Map<number, ISystemParameter>())

	handler(result: ISystemParameter[]): void {
		this.records = result;
		const newMap = new Map();
		for (const rec of result) {
			newMap.set(rec.ParameterID, rec);
		}
		this.recordsMap = newMap;
	}

	constructor() {
		super();
		this.fetch();
	}
}
