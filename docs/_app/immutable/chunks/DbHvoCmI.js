import { L as delegate, M as append, P as from_html, Q as sibling, W as template_effect, X as child, _t as reset, at as state, f as bind_this, j as set_text, rt as set, z as get } from "./CTB4HzdN.js";
import { t as Popover2 } from "./DnSz4jvr.js";
var root_1 = from_html(`<div class="popover2-container"><div class="popover2-content">This popover prefers top placement but will flip if there's no space above.</div></div>`);
var root_2 = from_html(`<div class="popover2-container"><div class="popover2-content">This popover prefers bottom placement but will flip if there's no space below.</div></div>`);
var root_3 = from_html(`<div class="popover2-container"><div class="popover2-content">This popover prefers left placement.</div></div>`);
var root_4 = from_html(`<div class="popover2-container"><div class="popover2-content">This popover prefers right placement.</div></div>`);
var root_5 = from_html(`<div class="popover2-container"><div class="popover2-content"><strong>Smart Positioning!</strong> <br/> This popover automatically finds the best position based on available space.
						Try scrolling the page or resizing the window. <br/> <br/> Current: <strong> </strong></div></div>`);
var root = from_html(`<div class="p-8 aL4 mt-[50vh] aMm"><div class="aMm"><h2 class="text-2xl font-bold mb-2">Popover2 Library</h2> <p class="text-gray-600 dark:text-gray-400">A simple library that renders elements in the body with smart positioning based on available space.</p></div> <div class="aL5 aMm"><h3 class="text-xl font-semibold aMm">Fixed Placement (will flip if no space)</h3> <div class="aL6 aL7 aL8 aMm"><div><button class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Top Placement</button> <!></div> <div><button class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600">Bottom Placement</button> <!></div> <div><button class="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600">Left Placement</button> <!></div> <div><button class="px-4 py-2 bg-orange-500 text-white rounded hover:bg-orange-600">Right Placement</button> <!></div></div></div> <div class="aL5 aMm"><h3 class="text-xl font-semibold aMm">Auto Placement (smart positioning)</h3> <p class="text-sm text-gray-600 dark:text-gray-400 aMm">Current placement: <strong> </strong></p> <div class="aMm"><button class="px-4 py-2 bg-indigo-500 text-white rounded hover:bg-indigo-600">Auto Placement</button> <!></div></div> <div class="aL5 aMm"><h3 class="text-xl font-semibold aMm">Scroll Test</h3> <p class="text-sm text-gray-600 dark:text-gray-400 aMm">Click the button below, then scroll to see how the popover repositions.</p></div> <div class="h-[150vh] aMm"></div> <div class="sticky bottom-8 left-0 right-0 aL6 justify-center aMm"><button class="px-6 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 shadow-lg">Bottom Button (Test Scroll)</button></div></div>`);
function Popover2Example($$anchor) {
	let topButton = state(null);
	let bottomButton = state(null);
	let leftButton = state(null);
	let rightButton = state(null);
	let autoButton = state(null);
	let showTop = state(false);
	let showBottom = state(false);
	let showLeft = state(false);
	let showRight = state(false);
	let showAuto = state(false);
	let currentPlacement = state("bottom");
	var div = root();
	var div_1 = sibling(child(div), 2);
	var div_2 = sibling(child(div_1), 2);
	var div_3 = child(div_2);
	var button = child(div_3);
	button.__click = () => set(showTop, !get(showTop));
	bind_this(button, ($$value) => set(topButton, $$value), () => get(topButton));
	Popover2(sibling(button, 2), {
		get referenceElement() {
			return get(topButton);
		},
		get open() {
			return get(showTop);
		},
		placement: "top",
		children: ($$anchor$1, $$slotProps) => {
			append($$anchor$1, root_1());
		},
		$$slots: { default: true }
	});
	reset(div_3);
	var div_5 = sibling(div_3, 2);
	var button_1 = child(div_5);
	button_1.__click = () => set(showBottom, !get(showBottom));
	bind_this(button_1, ($$value) => set(bottomButton, $$value), () => get(bottomButton));
	Popover2(sibling(button_1, 2), {
		get referenceElement() {
			return get(bottomButton);
		},
		get open() {
			return get(showBottom);
		},
		placement: "bottom",
		children: ($$anchor$1, $$slotProps) => {
			append($$anchor$1, root_2());
		},
		$$slots: { default: true }
	});
	reset(div_5);
	var div_7 = sibling(div_5, 2);
	var button_2 = child(div_7);
	button_2.__click = () => set(showLeft, !get(showLeft));
	bind_this(button_2, ($$value) => set(leftButton, $$value), () => get(leftButton));
	Popover2(sibling(button_2, 2), {
		get referenceElement() {
			return get(leftButton);
		},
		get open() {
			return get(showLeft);
		},
		placement: "left",
		children: ($$anchor$1, $$slotProps) => {
			append($$anchor$1, root_3());
		},
		$$slots: { default: true }
	});
	reset(div_7);
	var div_9 = sibling(div_7, 2);
	var button_3 = child(div_9);
	button_3.__click = () => set(showRight, !get(showRight));
	bind_this(button_3, ($$value) => set(rightButton, $$value), () => get(rightButton));
	Popover2(sibling(button_3, 2), {
		get referenceElement() {
			return get(rightButton);
		},
		get open() {
			return get(showRight);
		},
		placement: "right",
		children: ($$anchor$1, $$slotProps) => {
			append($$anchor$1, root_4());
		},
		$$slots: { default: true }
	});
	reset(div_9);
	reset(div_2);
	reset(div_1);
	var div_11 = sibling(div_1, 2);
	var p = sibling(child(div_11), 2);
	var strong = sibling(child(p));
	var text = child(strong, true);
	reset(strong);
	reset(p);
	var div_12 = sibling(p, 2);
	var button_4 = child(div_12);
	button_4.__click = () => set(showAuto, !get(showAuto));
	bind_this(button_4, ($$value) => set(autoButton, $$value), () => get(autoButton));
	Popover2(sibling(button_4, 2), {
		get referenceElement() {
			return get(autoButton);
		},
		get open() {
			return get(showAuto);
		},
		placement: "bottom",
		onPositionUpdate: (pos) => set(currentPlacement, pos.placement, true),
		children: ($$anchor$1, $$slotProps) => {
			var div_13 = root_5();
			var div_14 = child(div_13);
			var strong_1 = sibling(child(div_14), 8);
			var text_1 = child(strong_1, true);
			reset(strong_1);
			reset(div_14);
			reset(div_13);
			template_effect(() => set_text(text_1, get(currentPlacement)));
			append($$anchor$1, div_13);
		},
		$$slots: { default: true }
	});
	reset(div_12);
	reset(div_11);
	var div_15 = sibling(div_11, 6);
	var button_5 = child(div_15);
	button_5.__click = () => set(showAuto, !get(showAuto));
	reset(div_15);
	reset(div);
	template_effect(() => set_text(text, get(currentPlacement)));
	append($$anchor, div);
}
delegate(["click"]);
export { Popover2Example as t };
