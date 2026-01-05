import { M as append, P as from_html, Q as sibling, W as template_effect, X as child, Z as first_child, _t as reset, at as state, j as set_text, lt as pop, rt as set, u as init, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { m as GetHandler } from "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import "../chunks/DOFkf9MZ.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import "../chunks/BGbcPCgU.js";
import { t as ImageUploader } from "../chunks/5nFXSVM4.js";
var DemoService = class extends GetHandler {
	isTest = true;
	#message = state("");
	get message() {
		return get(this.#message);
	}
	set message(value) {
		set(this.#message, value, true);
	}
	route = "ruta de prueba";
	handler(e) {
		this.message = e.message;
	}
	constructor() {
		super();
		this.Test();
	}
};
var root_1 = from_html(`<div>hola</div> <div> </div> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, false);
	let demoService = new DemoService();
	init();
	Page($$anchor, {
		title: "demo",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = sibling(first_child(fragment_1), 2);
			var text = child(div, true);
			reset(div);
			ImageUploader(sibling(div, 2), {
				saveAPI: "producto-image",
				clearOnUpload: true,
				cardCss: "w-200 h-200 p-4",
				setDataToSend: (e) => {
					e.ProductoID = 8;
				},
				onUploaded: (src) => {
					console.log("imagen subida::", src);
				}
			});
			template_effect(() => set_text(text, demoService.message));
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
export { _page as component };
