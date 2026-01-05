import { $ as proxy, A as if_block, G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, R as event, S as set_class, W as template_effect, X as child, _t as reset, a as prop, at as state, gt as next, j as set_text, lt as pop, n as onDestroy, rt as set, st as user_derived, ut as push, v as set_attribute, x as set_style, z as get } from "./CTB4HzdN.js";
import { g as POST_XMLHR, s as imagesToUpload, y as Env } from "./BwrZ3UQO.js";
import { i as fileToImage, r as Notify } from "./DOFkf9MZ.js";
var root_1 = from_html(`<div class="w-full h-full relative flex flex-col items-center justify-center aJK aMv"><input type="file" accept="image/png, image/jpeg, image/webp" class="aMv"/> <div style="font-size: 2.4rem"><i class="icon-upload"></i></div> <div class="h5">Subir Imagen</div></div>`);
var root_3 = from_html(`<source type="image/avif"/>`);
var root_4 = from_html(`<source type="image/webp"/>`);
var root_2 = from_html(`<picture class="contents"><!> <!> <img class="w-full h-full absolute aJC aMv" alt="Upload preview"/></picture>`);
var root_6 = from_html(`<textarea class="w-full aJJ aMv" placeholder="Nombre..."></textarea>`);
var root_7 = from_html(`<div class="absolute w-full aJH aMv"> </div>`);
var root_8 = from_html(`<button class="bnr-1 _5 aJF aMv" aria-label="Subir imagen"><i class="icon-ok"></i></button>`);
var root_5 = from_html(`<div><!> <!> <div class="w-full absolute flex items-center justify-center aJI aMv"><button aria-label="Eliminar imagen"><i class="icon-cancel"></i></button> <!></div></div>`);
var root_9 = from_html(`<div class="w-full h-full absolute flex flex-col items-center justify-center aJG aMv"><div class="c-white h3 ff-bold">Loading...</div> <div class="flex relative items-center justify-center h-22 lh-10 w-[calc(100%-16px)] left-0 right-0 mt-8 _8 mr-8 ml-9 p-2 aMv"><div class="absolute _9 left-2 h-18 aMv"></div> <div class="absolute fs14 ff-bold text-white"> </div></div></div>`);
var root = from_html(`<div><!> <!> <!> <!></div>`);
function ImageUploader($$anchor, $$props) {
	push($$props, true);
	let src = prop($$props, "src", 3, ""), types = prop($$props, "types", 19, () => []), saveAPI = prop($$props, "saveAPI", 3, "images"), onUploaded = prop($$props, "onUploaded", 3, void 0), setDataToSend = prop($$props, "setDataToSend", 3, void 0), clearOnUpload = prop($$props, "clearOnUpload", 3, false), description = prop($$props, "description", 3, ""), cardStyle = prop($$props, "cardStyle", 3, ""), onDelete = prop($$props, "onDelete", 3, void 0), cardCss = prop($$props, "cardCss", 3, ""), hideFormUseMessage = prop($$props, "hideFormUseMessage", 3, ""), hideForm = prop($$props, "hideForm", 3, false), hideUploadButton = prop($$props, "hideUploadButton", 3, false), id = prop($$props, "id", 3, void 0);
	let imageSrc = state(proxy($$props.imageSource || {
		src: src(),
		types: types(),
		description: description()
	}));
	let progress = user_derived(() => src() || get(imageSrc)?.base64 ? -1 : 0);
	Env.imageCounter++;
	const imageID = id() || Env.imageCounter;
	user_effect(() => {
		src();
		$$props.imageSource;
		set(imageSrc, $$props.imageSource || {
			src: src(),
			base64: "",
			types: types(),
			description: description()
		}, true);
	});
	const makeImageSrc = (format) => {
		if (get(imageSrc).base64) return get(imageSrc).base64;
		if (!get(imageSrc).src) return "";
		let srcUrl = get(imageSrc).src || "";
		if (srcUrl.substring(0, 8) !== "https://" && srcUrl.substring(0, 7) !== "http://") {
			if ($$props.folder) srcUrl = $$props.folder + "/" + srcUrl;
			if ($$props.size) srcUrl = `${srcUrl}-x${$$props.size}`;
			srcUrl = Env.S3_URL + srcUrl;
		}
		if (format) srcUrl = srcUrl + "." + format;
		return srcUrl;
	};
	let imageFile;
	const uploadImage = async () => {
		let result = {};
		if (!get(imageSrc).base64) {
			Notify.failure("No hay nada que enviar.");
			return Promise.resolve(result);
		}
		set(progress, .001);
		const data = {
			Content: "",
			Content_x6: "",
			Content_x4: "",
			Content_x2: "",
			Description: get(imageSrc).description
		};
		if ($$props.useConvertAvif) {
			const resolutions = [
				{
					i: 6,
					r: 980,
					fn: (e) => data.Content_x6 = e
				},
				{
					i: 4,
					r: 670,
					fn: (e) => data.Content_x4 = e
				},
				{
					i: 2,
					r: 360,
					fn: (e) => data.Content_x2 = e
				}
			];
			for (const rs of resolutions) rs.promise = new Promise((resolve) => {
				fileToImage(imageFile, rs.r, "avif").then((d) => {
					rs.fn(d), resolve(0);
				});
			});
			try {
				await Promise.all(resolutions.map((x) => x.promise));
			} catch (error) {
				Notify.failure(`Error al convertir imagen: ${error}`);
				return Promise.resolve(result);
			}
		} else data.Content = get(imageSrc).base64;
		if (setDataToSend()) setDataToSend()(data);
		console.log("data a enviar::", data);
		try {
			result = await POST_XMLHR({
				data,
				route: saveAPI() || "images",
				onUploadProgress: (e) => {
					if (e.total) {
						set(progress, Math.round(e.loaded * 100 / e.total));
						console.log("progress:: ", get(progress));
					}
				}
			});
		} catch (error) {
			Notify.failure("Error guardando la imagen: " + String(error));
			return result;
		}
		result.id = imageID;
		result.description = get(imageSrc).description;
		if (clearOnUpload()) set(imageSrc, {
			src: "",
			base64: "",
			types: [],
			description: ""
		}, true);
		else set(imageSrc, {
			src: `${result.imageName}-x2`,
			base64: "",
			types: ["webp", "avif"],
			description: get(imageSrc).description
		}, true);
		set(progress, -1);
		if (onUploaded()) onUploaded()(result.imageName, result.description);
		return result;
	};
	const onFileChange = async (ev) => {
		const files = ev.target.files;
		if (!files || files.length === 0) return;
		imageFile = files[0];
		console.log("imagefile::", imageFile);
		try {
			const imageB64 = await fileToImage(imageFile, 1200, "avif");
			console.log("imageB64", imageB64);
			set(progress, -1);
			set(imageSrc, {
				src: "",
				base64: imageB64,
				types: [],
				description: get(imageSrc).description
			}, true);
			$$props.onChange?.(get(imageSrc), uploadImage);
		} catch (error) {
			Notify.failure("Error procesando la imagen: " + String(error));
			set(progress, 0);
		}
	};
	user_effect(() => {
		if (get(imageSrc).base64) imagesToUpload.set(imageID, uploadImage);
		else imagesToUpload.delete(imageID);
	});
	onDestroy(() => {
		imagesToUpload.delete(imageID);
	});
	var div = root();
	var node = child(div);
	var consequent = ($$anchor$1) => {
		var div_1 = root_1();
		var input = child(div_1);
		input.__change = onFileChange;
		next(4);
		reset(div_1);
		append($$anchor$1, div_1);
	};
	if_block(node, ($$render) => {
		if ((get(imageSrc)?.src || "").length === 0) $$render(consequent);
	});
	var node_1 = sibling(node, 2);
	var consequent_3 = ($$anchor$1) => {
		var picture = root_2();
		var node_2 = child(picture);
		var consequent_1 = ($$anchor$2) => {
			var source = root_3();
			template_effect(($0) => set_attribute(source, "srcset", $0), [() => makeImageSrc("avif")]);
			append($$anchor$2, source);
		};
		if_block(node_2, ($$render) => {
			if (get(imageSrc).types?.includes("avif")) $$render(consequent_1);
		});
		var node_3 = sibling(node_2, 2);
		var consequent_2 = ($$anchor$2) => {
			var source_1 = root_4();
			template_effect(($0) => set_attribute(source_1, "srcset", $0), [() => makeImageSrc("webp")]);
			append($$anchor$2, source_1);
		};
		if_block(node_3, ($$render) => {
			if (get(imageSrc).types?.includes("webp")) $$render(consequent_2);
		});
		var img = sibling(node_3, 2);
		reset(picture);
		template_effect(($0) => set_attribute(img, "src", $0), [() => makeImageSrc(get(imageSrc)?.types?.[0])]);
		append($$anchor$1, picture);
	};
	if_block(node_1, ($$render) => {
		if ((get(imageSrc).src || get(imageSrc).base64 || "").length > 0) $$render(consequent_3);
	});
	var node_4 = sibling(node_1, 2);
	var consequent_7 = ($$anchor$1) => {
		var div_2 = root_5();
		var node_5 = child(div_2);
		var consequent_4 = ($$anchor$2) => {
			var textarea = root_6();
			set_attribute(textarea, "rows", 3);
			event("blur", textarea, (ev) => {
				ev.stopPropagation();
				get(imageSrc).description = ev.currentTarget.value || "";
				$$props.onChange?.(get(imageSrc), uploadImage);
			});
			append($$anchor$2, textarea);
		};
		if_block(node_5, ($$render) => {
			if (get(imageSrc).base64 && !hideFormUseMessage() && !hideForm()) $$render(consequent_4);
		});
		var node_6 = sibling(node_5, 2);
		var consequent_5 = ($$anchor$2) => {
			var div_3 = root_7();
			var text = child(div_3, true);
			reset(div_3);
			template_effect(() => set_text(text, hideFormUseMessage()));
			append($$anchor$2, div_3);
		};
		if_block(node_6, ($$render) => {
			if (get(imageSrc).base64 && hideFormUseMessage()) $$render(consequent_5);
		});
		var div_4 = sibling(node_6, 2);
		var button = child(div_4);
		button.__click = (ev) => {
			ev.stopPropagation();
			if (onDelete()) {
				onDelete()(src());
				return;
			}
			set(imageSrc, {
				src: "",
				base64: "",
				types: [],
				description: ""
			}, true);
			$$props.onChange?.(get(imageSrc));
			set(progress, 0);
		};
		var node_7 = sibling(button, 2);
		var consequent_6 = ($$anchor$2) => {
			var button_1 = root_8();
			button_1.__click = (ev) => {
				ev.stopPropagation();
				uploadImage();
			};
			append($$anchor$2, button_1);
		};
		if_block(node_7, ($$render) => {
			if (get(imageSrc).base64 && !hideUploadButton()) $$render(consequent_6);
		});
		reset(div_4);
		reset(div_2);
		template_effect(() => {
			set_class(div_2, 1, `w-full h-full absolute aJD${get(imageSrc).base64 ? " s1" : ""}`, "aMv");
			set_class(button, 1, `bnr-1 _4 mr-12 ${get(imageSrc).base64 ? "" : "aJE"} aJF`, "aMv");
		});
		append($$anchor$1, div_2);
	};
	if_block(node_4, ($$render) => {
		if (get(progress) === -1) $$render(consequent_7);
	});
	var node_8 = sibling(node_4, 2);
	var consequent_8 = ($$anchor$1) => {
		var div_5 = root_9();
		var div_6 = sibling(child(div_5), 2);
		var div_7 = child(div_6);
		var div_8 = sibling(div_7, 2);
		var text_1 = child(div_8);
		reset(div_8);
		reset(div_6);
		reset(div_5);
		template_effect(($0, $1) => {
			set_style(div_7, `width: calc(${$0 ?? ""}% - 4px);`);
			set_text(text_1, `${$1 ?? ""} %`);
		}, [() => Math.round(get(progress)), () => Math.round(get(progress))]);
		append($$anchor$1, div_5);
	};
	if_block(node_8, ($$render) => {
		if (get(progress) > 0) $$render(consequent_8);
	});
	reset(div);
	template_effect(() => {
		set_class(div, 1, `relative ${cardCss() ?? ""} aJz ${get(imageSrc)?.src ? "" : "aJB"}`, "aMv");
		set_style(div, cardStyle());
	});
	append($$anchor, div);
	pop();
}
delegate(["change", "click"]);
export { ImageUploader as t };
