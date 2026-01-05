import { $ as proxy, A as if_block, C as clsx, D as html, F as text, G as user_effect, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, R as event, S as set_class, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, f as bind_this, j as set_text, k as index, lt as pop, rt as set, st as user_derived, ut as push, v as set_attribute, vt as noop, x as set_style, z as get } from "./CTB4HzdN.js";
import { t as Core } from "./BwrZ3UQO.js";
import { a as throttle } from "./DwKmqXYU.js";
import { o as highlString, s as include } from "./DOFkf9MZ.js";
import { t as components_module_default } from "./cyfMRGwx.js";
import { n as dist_default } from "./BCJF-hdW.js";
var root_1 = from_html(`<div><div></div></div> <div> <!></div> <div><div></div></div> <div><div></div></div>`, 1);
var root_2 = from_html(`<div><div></div></div>`);
var root_3 = from_html(`<input/>`);
var root_6 = from_html(`<div class="fs15 mt-2 _10 aMp"> </div>`);
var root_4 = from_html(`<div role="button" tabindex="0"><div class="h100 w100 flex items-center"><!></div></div>`);
var root_8 = from_html(`<div>Cargando...</div>`);
var root_9 = from_html(`<div><i></i></div>`);
var root_12 = from_html(`<span> </span>`);
var root_11 = from_html(`<div role="option" tabindex="0"><div></div></div>`);
var root_10 = from_html(`<div role="button" tabindex="0"><!></div>`);
var root = from_html(`<div><!> <div><!> <!> <!></div> <!> <!> <!></div>`);
function SearchSelect($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15), css = prop($$props, "css", 3, ""), options = prop($$props, "options", 19, () => []);
	prop($$props, "max", 3, 100);
	const notEmpty = prop($$props, "notEmpty", 3, false), required = prop($$props, "required", 3, false), disabled = prop($$props, "disabled", 3, false), clearOnSelect = prop($$props, "clearOnSelect", 3, false), avoidIDs = prop($$props, "avoidIDs", 19, () => []), inputCss = prop($$props, "inputCss", 3, ""), showLoading = prop($$props, "showLoading", 3, false);
	let show = state(false);
	let filteredOptions = state(proxy([...options()]));
	let arrowSelected = state(-1);
	let avoidhover = state(false);
	let isValid = state(0);
	let selectedValue = state("");
	let avoidBlur = false;
	let openUp = state(false);
	let inputRef = state(void 0);
	let vlRef = state(void 0);
	let words = state(proxy([]));
	const isMobile = user_derived(() => Core.deviceType === 3);
	const useLayerPicker = user_derived(() => get(isMobile));
	const isDisabled = user_derived(() => disabled() || showLoading());
	function checkPosition() {
		if (get(inputRef)) {
			const rect = get(inputRef).getBoundingClientRect();
			const windowHeight = window.innerHeight;
			set(openUp, rect.top > windowHeight / 2);
		}
	}
	function changeValue(value) {
		if (get(useLayerPicker)) set(selectedValue, value, true);
		else if (get(inputRef)) get(inputRef).value = value;
	}
	function getSelectedFromProps() {
		let currValue = $$props.selected;
		if (typeof currValue === "undefined" && $$props.save && saveOn()) currValue = saveOn()[$$props.save];
		let selectedItem;
		if (currValue) selectedItem = options().find((x) => x[$$props.keyId] === currValue);
		return selectedItem;
	}
	function isRequired() {
		return required() && !disabled();
	}
	let prevSetValueTime = Date.now();
	const setValueSaveOn = (selectedItem, setOnInput) => {
		const nowTime = Date.now();
		if (nowTime - prevSetValueTime < 80) return;
		prevSetValueTime = nowTime;
		if (notEmpty() && !selectedItem) selectedItem = getSelectedFromProps();
		if (setOnInput) if (selectedItem && !clearOnSelect()) changeValue(selectedItem[$$props.keyName]);
		else changeValue("");
		const newValue = selectedItem ? selectedItem[$$props.keyId] : null;
		if (isRequired()) set(isValid, newValue ? 2 : 1, true);
		if (clearOnSelect()) {
			if ($$props.onChange) $$props.onChange(selectedItem);
		} else if (saveOn() && $$props.save) {
			if ((saveOn()[$$props.save] || null) !== newValue) {
				saveOn(saveOn()[$$props.save] = newValue, true);
				if ($$props.onChange) $$props.onChange(selectedItem);
			}
		} else if (typeof $$props.selected !== "undefined") {
			if (($$props.selected || null) !== newValue) {
				if ($$props.onChange) $$props.onChange(selectedItem);
			}
		}
	};
	const filter = (text$1) => {
		console.log("options filtered::", $$props.label, options());
		if (!text$1 && (avoidIDs()?.length || 0) === 0) return [...options()];
		const avoidIDSet = new Set(avoidIDs() || []);
		const filtered = [];
		const searchWords = text$1.toLowerCase().split(" ").filter((x) => x.length > 1);
		for (const opt of options()) {
			if (avoidIDSet.size > 0 && avoidIDSet.has(opt[$$props.keyId])) continue;
			if (searchWords.length === 0) filtered.push(opt);
			else {
				const name = opt[$$props.keyName];
				if (typeof name === "string") {
					if (include(name.toLowerCase(), searchWords)) filtered.push(opt);
				}
			}
		}
		console.log("options filtered 2::", $$props.label, options(), filtered);
		return filtered;
	};
	function onKeyUp(ev) {
		ev.stopPropagation();
		const target = ev.target;
		const text$1 = String(target.value || "").toLowerCase();
		throttle(() => {
			set(words, String(get(inputRef).value).toLowerCase().split(" "), true);
			set(filteredOptions, filter(text$1), true);
			set(arrowSelected, -1);
		}, 120);
	}
	function onKeyDown(ev) {
		console.log("avoid hover:: ", get(avoidhover));
		ev.stopPropagation();
		if (!get(show) || get(filteredOptions).length === 0) return;
		if (ev.key === "ArrowUp") {
			ev.preventDefault();
			set(arrowSelected, get(arrowSelected) <= 0 ? get(filteredOptions).length - 1 : get(arrowSelected) - 1, true);
			set(avoidhover, true);
			get(vlRef)?.scrollToIndex(get(arrowSelected), { align: "auto" });
		} else if (ev.key === "ArrowDown") {
			ev.preventDefault();
			set(arrowSelected, get(arrowSelected) >= get(filteredOptions).length - 1 ? 0 : get(arrowSelected) + 1, true);
			set(avoidhover, true);
			get(vlRef)?.scrollToIndex(get(arrowSelected), { align: "auto" });
		} else if (ev.key === "Enter" && get(arrowSelected) >= 0) {
			ev.preventDefault();
			onOptionClick(get(filteredOptions)[get(arrowSelected)]);
		}
	}
	function onOptionClick(opt) {
		setValueSaveOn(opt, true);
		if (get(inputRef)) get(inputRef).blur();
		set(show, false);
	}
	let cN = user_derived(() => `${components_module_default.input} p-rel${css() ? ` ${css()}` : ""}${!$$props.label ? " no-label" : ""}`);
	function iconValid() {
		if (!get(isValid)) return null;
		if (get(isValid) === 2) return `<i class="v-icon icon-ok c-green"></i>`;
		else if (get(isValid) === 1) return `<i class="v-icon icon-attention c-red"></i>`;
		return null;
	}
	user_effect(() => {
		const selectedItem = getSelectedFromProps();
		if (selectedItem) changeValue(selectedItem[$$props.keyName]);
		else changeValue("");
		if (isRequired()) set(isValid, selectedItem ? 2 : 1, true);
	});
	user_effect(() => {
		set(filteredOptions, filter(""), true);
		set(arrowSelected, -1);
	});
	const handleOpenMobileLayer = () => {
		Core.showMobileSearchLayer = {
			options: get(filteredOptions),
			keyName: $$props.keyName,
			keyID: $$props.keyId,
			onSelect: (e) => {
				onOptionClick(e);
			},
			onRemove: (e) => {}
		};
	};
	var div = root();
	var node = child(div);
	var consequent = ($$anchor$1) => {
		var fragment = root_1();
		var div_1 = first_child(fragment);
		var div_2 = sibling(div_1, 2);
		var text_1 = child(div_2, true);
		html(sibling(text_1), () => iconValid() || "");
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var div_4 = sibling(div_3, 2);
		template_effect(() => {
			set_class(div_1, 1, clsx(components_module_default.input_lab_cell_left), "aMp");
			set_class(div_2, 1, clsx(components_module_default.input_lab), "aMp");
			set_text(text_1, $$props.label);
			set_class(div_3, 1, clsx(components_module_default.input_lab_cell_right), "aMp");
			set_class(div_4, 1, clsx(components_module_default.input_shadow_layer), "aMp");
		});
		append($$anchor$1, fragment);
	};
	if_block(node, ($$render) => {
		if ($$props.label) $$render(consequent);
	});
	var div_5 = sibling(node, 2);
	var node_2 = child(div_5);
	var consequent_1 = ($$anchor$1) => {
		var div_6 = root_2();
		template_effect(() => set_class(div_6, 1, clsx(components_module_default.input_div_1), "aMp"));
		append($$anchor$1, div_6);
	};
	if_block(node_2, ($$render) => {
		if ($$props.label) $$render(consequent_1);
	});
	var node_3 = sibling(node_2, 2);
	var consequent_2 = ($$anchor$1) => {
		var input = root_3();
		input.__keyup = onKeyUp;
		input.__keydown = (ev) => {
			ev.stopPropagation();
			console.log(ev);
			onKeyDown(ev);
		};
		bind_this(input, ($$value) => set(inputRef, $$value), () => get(inputRef));
		template_effect(() => {
			set_class(input, 1, `w-full ${components_module_default.input_inp} ${inputCss()}`, "aMp");
			set_attribute(input, "placeholder", showLoading() ? "" : $$props.placeholder || ":: seleccione ::");
			input.disabled = get(isDisabled) || get(isMobile);
		});
		event("paste", input, onKeyUp);
		event("cut", input, onKeyUp);
		event("focus", input, (ev) => {
			ev.stopPropagation();
			if (!get(show)) {
				checkPosition();
				set(words, [], true);
				set(filteredOptions, filter(""), true);
				set(show, true);
			}
		});
		event("blur", input, (ev) => {
			ev.stopPropagation();
			console.log("avoidBlur 2", avoidBlur);
			if (avoidBlur) {
				avoidBlur = false;
				get(inputRef).focus();
				return;
			}
			let inputValue = String(get(inputRef).value || "").toLowerCase();
			setValueSaveOn(options().find((x) => {
				return String(x[$$props.keyName] || "").toLowerCase() === inputValue;
			}), true);
			set(show, false);
		});
		append($$anchor$1, input);
	};
	var alternate_1 = ($$anchor$1) => {
		var div_7 = root_4();
		div_7.__keydown = (ev) => {
			if (ev.key === "Enter" || ev.key === " ") handleOpenMobileLayer();
		};
		div_7.__click = (ev) => {
			ev.stopPropagation();
			handleOpenMobileLayer();
		};
		var div_8 = child(div_7);
		var node_4 = child(div_8);
		var consequent_3 = ($$anchor$2) => {
			var text_2 = text();
			template_effect(() => set_text(text_2, get(selectedValue)));
			append($$anchor$2, text_2);
		};
		var alternate = ($$anchor$2) => {
			var div_9 = root_6();
			var text_3 = child(div_9, true);
			reset(div_9);
			template_effect(() => set_text(text_3, $$props.placeholder || ""));
			append($$anchor$2, div_9);
		};
		if_block(node_4, ($$render) => {
			if (get(selectedValue)) $$render(consequent_3);
			else $$render(alternate, false);
		});
		reset(div_8);
		reset(div_7);
		template_effect(() => set_class(div_7, 1, `w-full flex items-center ${components_module_default.input_inp} ${inputCss()}`, "aMp"));
		append($$anchor$1, div_7);
	};
	if_block(node_3, ($$render) => {
		if (!get(useLayerPicker)) $$render(consequent_2);
		else $$render(alternate_1, false);
	});
	var node_5 = sibling(node_3, 2);
	var consequent_4 = ($$anchor$1) => {
		var fragment_2 = comment();
		html(first_child(fragment_2), () => iconValid() || "");
		append($$anchor$1, fragment_2);
	};
	if_block(node_5, ($$render) => {
		if (!$$props.label) $$render(consequent_4);
	});
	reset(div_5);
	var node_7 = sibling(div_5, 2);
	var consequent_5 = ($$anchor$1) => {
		append($$anchor$1, root_8());
	};
	if_block(node_7, ($$render) => {
		if (showLoading()) $$render(consequent_5);
	});
	var node_8 = sibling(node_7, 2);
	var consequent_6 = ($$anchor$1) => {
		var div_11 = root_9();
		var i_1 = child(div_11);
		reset(div_11);
		template_effect(() => {
			set_class(div_11, 1, `absolute bottom-8 right-6 pointer-events-none ${get(show) && !$$props.icon ? "show" : ""}`);
			set_class(i_1, 1, clsx($$props.icon || "icon-down-open-1"), "aMp");
		});
		append($$anchor$1, div_11);
	};
	if_block(node_8, ($$render) => {
		if (!get(isDisabled)) $$render(consequent_6);
	});
	var node_9 = sibling(node_8, 2);
	var consequent_7 = ($$anchor$1) => {
		var div_12 = root_10();
		let classes;
		div_12.__mousedown = (ev) => {
			ev.stopPropagation();
			avoidBlur = true;
			console.log("avoidBlur 1", avoidBlur);
		};
		div_12.__mousemove = function(...$$args) {
			(get(avoidhover) ? (ev) => {
				console.log("hover aqui:: ", get(arrowSelected));
				ev.stopPropagation();
				if (get(avoidhover)) {
					set(arrowSelected, -1);
					set(avoidhover, false);
				}
			} : void 0)?.apply(this, $$args);
		};
		let styles;
		var node_10 = child(div_12);
		{
			const renderItem = ($$anchor$2, e = noop, i = noop) => {
				const name = user_derived(() => String(e()[$$props.keyName]));
				const highlighted = user_derived(() => highlString(get(name), get(words)));
				var div_13 = root_11();
				div_13.__click = (ev) => {
					ev.stopPropagation();
					onOptionClick(e());
				};
				var div_14 = child(div_13);
				each(div_14, 21, () => get(highlighted), index, ($$anchor$3, w) => {
					var span = root_12();
					let classes_1;
					var text_4 = child(span, true);
					reset(span);
					template_effect(() => {
						classes_1 = set_class(span, 1, clsx(get(w).highl ? "_8" : ""), "aMp", classes_1, { "mr-4": get(w).isEnd });
						set_text(text_4, get(w).text);
					});
					append($$anchor$3, span);
				});
				reset(div_14);
				reset(div_13);
				template_effect(() => {
					set_class(div_13, 1, `flex ai-center aJU${get(arrowSelected) === i() ? " aJV" : ""}`, "aMp");
					set_attribute(div_13, "aria-selected", get(arrowSelected) === i());
				});
				append($$anchor$2, div_13);
			};
			bind_this(dist_default(node_10, {
				get items() {
					return get(filteredOptions);
				},
				renderItem,
				$$slots: { renderItem: true }
			}), ($$value) => set(vlRef, $$value, true), () => get(vlRef));
		}
		reset(div_12);
		template_effect(($0) => {
			classes = set_class(div_12, 1, `_1 p-4 left-0 z-320 ${get(arrowSelected) >= 0 ? " on-arrow" : ""} ${($$props.optionsCss || "w-full") ?? ""}`, "aMp", classes, { "open-up": get(openUp) });
			styles = set_style(div_12, "", styles, $0);
		}, [() => ({ height: Math.min(get(filteredOptions).length * 36 + 10, 300) + "px" })]);
		append($$anchor$1, div_12);
	};
	if_block(node_9, ($$render) => {
		if (get(show) && !get(useLayerPicker)) $$render(consequent_7);
	});
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(get(cN)), "aMp");
		set_class(div_5, 1, `${components_module_default.input_div} flex w-full`, "aMp");
	});
	append($$anchor, div);
	pop();
}
delegate([
	"keyup",
	"keydown",
	"click",
	"mousedown",
	"mousemove"
]);
export { SearchSelect as t };
