import { A as if_block, C as clsx, E as snippet, F as text, G as user_effect, L as delegate, M as append, N as comment, P as from_html, Q as sibling, S as set_class, V as untrack, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, f as bind_this, j as set_text, lt as pop, rt as set, st as user_derived, ut as push, v as set_attribute, vt as noop, z as get } from "./CTB4HzdN.js";
import { i as closeModal, l as openModals } from "./BwrZ3UQO.js";
import { n as Portal } from "./DnSz4jvr.js";
var root_5 = from_html(`<button class="bx-red mr-10 lh-10" aria-label="Eliminar"><i class="icon-trash"></i></button>`);
var root_6 = from_html(`<button class="bx-blue mr-10 lh-10"><i></i> <span class="_5 aMk"> </span></button>`);
var root_2 = from_html(`<div class="_1 fixed top-0 left-0 flex items-center justify-center aMk"><div><div><div class="flex items-center ff-bold leading-[1.1] text-lg md:text-xl"><!></div> <div class="flex items-center"><!> <!> <button class="bx-yellow h3 lh-10 mr-[-2px]" aria-label="Cerrar"><i class="icon-cancel"></i></button></div></div> <div><!></div></div></div>`);
function Modal($$anchor, $$props) {
	push($$props, true);
	let css = prop($$props, "css", 3, ""), isEdit = prop($$props, "isEdit", 3, false);
	let isOpen = state(false);
	let modalDiv = state(void 0);
	user_effect(() => {
		const isThisModalOpen = openModals.includes($$props.id);
		if (get(isOpen) === isThisModalOpen) return;
		if (isThisModalOpen) {
			set(isOpen, true);
			untrack(() => {
				setTimeout(() => {
					if (get(modalDiv)) get(modalDiv).classList.add("modal-show");
				}, 0);
			});
		} else untrack(() => {
			get(modalDiv)?.classList?.remove("modal-show");
			setTimeout(() => {
				set(isOpen, false);
			}, 300);
		});
	});
	function handleClose(ev) {
		ev.stopPropagation();
		if ($$props.onClose) $$props.onClose();
		closeModal($$props.id);
	}
	function handleDelete(ev) {
		if ($$props.onDelete) {
			$$props.onDelete();
			ev.stopPropagation();
		}
	}
	function handleSave(ev) {
		if ($$props.onSave) {
			$$props.onSave();
			ev.stopPropagation();
		}
	}
	function isSnippet(value) {
		return typeof value === "function";
	}
	const modalSizesMap = new Map([
		[1, "w-600 max-w-[64vw]"],
		[2, "w-650 max-w-[66vw]"],
		[3, "w-700 max-w-[68vw]"],
		[4, "w-750 max-w-[72vw]"],
		[5, "w-800 max-w-[75vw]"],
		[6, "w-850 max-w-[78vw]"],
		[7, "w-900 max-w-[82vw]"],
		[8, "w-950 max-w-[84vw]"],
		[9, "w-1000 max-w-[88vw]"]
	]);
	const saveLabel = user_derived(() => {
		if ($$props.saveButtonLabel) return $$props.saveButtonLabel;
		return isEdit() ? "Actualizar" : "Guardar";
	});
	var fragment = comment();
	var node = first_child(fragment);
	var consequent_3 = ($$anchor$1) => {
		Portal($$anchor$1, {
			children: ($$anchor$2, $$slotProps) => {
				var div = root_2();
				var div_1 = child(div);
				var div_2 = child(div_1);
				var div_3 = child(div_2);
				var node_1 = child(div_3);
				var consequent = ($$anchor$3) => {
					var fragment_2 = comment();
					snippet(first_child(fragment_2), () => $$props.title);
					append($$anchor$3, fragment_2);
				};
				var alternate = ($$anchor$3) => {
					var text$1 = text();
					template_effect(() => set_text(text$1, $$props.title));
					append($$anchor$3, text$1);
				};
				if_block(node_1, ($$render) => {
					if (isSnippet($$props.title)) $$render(consequent);
					else $$render(alternate, false);
				});
				reset(div_3);
				var div_4 = sibling(div_3, 2);
				var node_3 = child(div_4);
				var consequent_1 = ($$anchor$3) => {
					var button = root_5();
					button.__click = handleDelete;
					append($$anchor$3, button);
				};
				if_block(node_3, ($$render) => {
					if ($$props.onDelete) $$render(consequent_1);
				});
				var node_4 = sibling(node_3, 2);
				var consequent_2 = ($$anchor$3) => {
					var button_1 = root_6();
					button_1.__click = handleSave;
					var i = child(button_1);
					var span = sibling(i, 2);
					var text_1 = child(span, true);
					reset(span);
					reset(button_1);
					template_effect(() => {
						set_attribute(button_1, "aria-label", get(saveLabel));
						set_class(i, 1, clsx($$props.saveIcon || "icon-floppy"), "aMk");
						set_text(text_1, get(saveLabel));
					});
					append($$anchor$3, button_1);
				};
				if_block(node_4, ($$render) => {
					if ($$props.onSave) $$render(consequent_2);
				});
				var button_2 = sibling(node_4, 2);
				button_2.__click = handleClose;
				reset(div_4);
				reset(div_2);
				var div_5 = sibling(div_2, 2);
				snippet(child(div_5), () => $$props.children ?? noop);
				reset(div_5);
				reset(div_1);
				reset(div);
				bind_this(div, ($$value) => set(modalDiv, $$value), () => get(modalDiv));
				template_effect(($0) => {
					set_class(div_1, 1, `_2 pt-50 min-h-460 flex flex-col relative ${css() ?? ""} ${$0 ?? ""}`, "aMk");
					set_class(div_2, 1, `_3 h-50 py-0 px-8 md:px-12 flex absolute w-full top-0 left-0 items-center justify-between mb-auto ${$$props.headCss ?? ""}`, "aMk");
					set_class(div_5, 1, `w-full grow py-6 px-2 md:px-10 ${$$props.bodyCss ?? ""}`, "aMk");
				}, [() => modalSizesMap.get($$props.size)]);
				append($$anchor$2, div);
			},
			$$slots: { default: true }
		});
	};
	if_block(node, ($$render) => {
		if (get(isOpen)) $$render(consequent_3);
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
export { Modal as t };
