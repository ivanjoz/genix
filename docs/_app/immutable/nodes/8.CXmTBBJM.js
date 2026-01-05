import { M as append, P as from_html } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import "../chunks/BGbcPCgU.js";
var root_1 = from_html(`<h2>hola mundo</h2>`);
function _page($$anchor) {
	Page($$anchor, {
		title: "Page Editor",
		children: ($$anchor$1, $$slotProps) => {
			append($$anchor$1, root_1());
		},
		$$slots: { default: true }
	});
}
export { _page as component };
