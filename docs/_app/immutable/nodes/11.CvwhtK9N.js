import { L as delegate, M as append, P as from_html, Q as sibling, X as child, _t as reset, at as state, f as bind_this, gt as next, rt as set, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/2yAUi6s0.js";
import "../chunks/DwKmqXYU.js";
import { t as Popover2 } from "../chunks/DnSz4jvr.js";
/* empty css                 */
var root_1 = from_html(`<div class="popover2-container"><div class="popover2-content"><strong>✅ Success!</strong> <br/> This popover is rendered in the document.body, so it escapes the overflow:hidden parent. <br/><br/> <button class="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-sm">Close</button></div></div>`);
var root_2 = from_html(`<div class="popover2-container"><div class="popover2-content"><strong>✅ Portal Magic!</strong> <br/> This button is inside 2 levels of overflow:hidden, but the popover still renders perfectly in the body. <br/><br/> <button class="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-sm">Close</button></div></div>`);
var root = from_html(`<div class="p-8"><h2 class="text-2xl font-bold mb-4">Overflow Hidden Test</h2> <p class="text-gray-600 dark:text-gray-400 mb-8">This tests that the popover works even inside containers with <code class="aMc">overflow: hidden</code>.
		Without proper portal rendering, the popover would be clipped.</p> <div class="border-2 border-red-500 p-4 mb-8" style="overflow: hidden; height: 200px;"><h3 class="text-lg font-semibold mb-2 text-red-600">❌ Container with overflow: hidden</h3> <p class="text-sm text-gray-600 mb-4">The popover should still appear correctly and NOT be clipped by this container.</p> <button class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Click me (inside overflow:hidden)</button> <!></div> <div class="border-2 border-purple-500 p-4" style="overflow: hidden; height: 250px;"><h3 class="text-lg font-semibold mb-2 text-purple-600">❌ Nested Container (also overflow: hidden)</h3> <div class="border-2 border-orange-500 p-4 mt-2" style="overflow: hidden; height: 150px;"><h4 class="text-md font-semibold mb-2 text-orange-600">❌ Double nested with overflow: hidden</h4> <p class="text-sm text-gray-600 mb-4">Even with multiple nested overflow:hidden containers, the popover works!</p> <button class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600">Deeply nested button</button> <!></div></div> <div class="border-2 border-green-500 p-4 mt-8"><h3 class="text-lg font-semibold mb-2 text-green-600">✅ Normal Container (no overflow hidden)</h3> <p class="text-sm text-gray-600">For comparison, this is a normal container without overflow issues.</p></div> <div class="mt-8 p-4 bg-blue-50 dark:bg-blue-900 rounded"><h3 class="font-semibold mb-2">🔍 How to verify:</h3> <ol class="list-decimal ml-5 space-y-1 text-sm"><li>Click the buttons inside the red and purple containers</li> <li>The popovers should appear fully visible, not clipped</li> <li>Open the browser DevTools and inspect the popover element</li> <li>You should see it's a direct child of <code class="aMc">&lt;body&gt;</code>, not inside the overflow containers</li> <li>Try scrolling and resizing - the popover stays positioned correctly</li></ol></div></div>`);
function _page($$anchor) {
	let button1 = state(null);
	let button2 = state(null);
	let show1 = state(false);
	let show2 = state(false);
	var div = root();
	var div_1 = sibling(child(div), 4);
	var button = sibling(child(div_1), 4);
	button.__click = () => set(show1, !get(show1));
	bind_this(button, ($$value) => set(button1, $$value), () => get(button1));
	Popover2(sibling(button, 2), {
		get referenceElement() {
			return get(button1);
		},
		get open() {
			return get(show1);
		},
		placement: "bottom",
		children: ($$anchor$1, $$slotProps) => {
			var div_2 = root_1();
			var div_3 = child(div_2);
			var button_1 = sibling(child(div_3), 7);
			button_1.__click = () => set(show1, false);
			reset(div_3);
			reset(div_2);
			append($$anchor$1, div_2);
		},
		$$slots: { default: true }
	});
	reset(div_1);
	var div_4 = sibling(div_1, 2);
	var div_5 = sibling(child(div_4), 2);
	var button_2 = sibling(child(div_5), 4);
	button_2.__click = () => set(show2, !get(show2));
	bind_this(button_2, ($$value) => set(button2, $$value), () => get(button2));
	Popover2(sibling(button_2, 2), {
		get referenceElement() {
			return get(button2);
		},
		get open() {
			return get(show2);
		},
		placement: "right",
		children: ($$anchor$1, $$slotProps) => {
			var div_6 = root_2();
			var div_7 = child(div_6);
			var button_3 = sibling(child(div_7), 7);
			button_3.__click = () => set(show2, false);
			reset(div_7);
			reset(div_6);
			append($$anchor$1, div_6);
		},
		$$slots: { default: true }
	});
	reset(div_5);
	reset(div_4);
	next(4);
	reset(div);
	append($$anchor, div);
}
delegate(["click"]);
export { _page as component };
