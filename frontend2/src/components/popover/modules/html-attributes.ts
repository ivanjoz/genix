export interface HTMLAttributes<T extends keyof HTMLElementTagNameMap> {
	class?: string;
	style?: string;
	id?: string;
	[key: string]: any;
}
