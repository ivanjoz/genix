import { bt as false_default, ct as getContext } from "./CTB4HzdN.js";
import { n as true_default } from "./D9PGjZ6Y.js";
import { a as page$3, i as navigating$1, o as updated$1, r as stores } from "./D25ese0K.js";
const page$2 = {
	get data() {
		return page$3.data;
	},
	get error() {
		return page$3.error;
	},
	get form() {
		return page$3.form;
	},
	get params() {
		return page$3.params;
	},
	get route() {
		return page$3.route;
	},
	get state() {
		return page$3.state;
	},
	get status() {
		return page$3.status;
	},
	get url() {
		return page$3.url;
	}
};
Object.defineProperty({
	get from() {
		return navigating$1.current ? navigating$1.current.from : null;
	},
	get to() {
		return navigating$1.current ? navigating$1.current.to : null;
	},
	get type() {
		return navigating$1.current ? navigating$1.current.type : null;
	},
	get willUnload() {
		return navigating$1.current ? navigating$1.current.willUnload : null;
	},
	get delta() {
		return navigating$1.current ? navigating$1.current.delta : null;
	},
	get complete() {
		return navigating$1.current ? navigating$1.current.complete : null;
	}
}, "current", { get() {
	throw new Error("Replace navigating.current.<prop> with navigating.<prop>");
} });
stores.updated.check;
const page = page$2;
export { page as t };
