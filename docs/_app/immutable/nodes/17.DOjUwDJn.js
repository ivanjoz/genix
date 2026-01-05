import { $ as proxy, L as delegate, M as append, P as from_html, Q as sibling, W as template_effect, X as child, _t as reset, at as state, lt as pop, r as onMount, rt as set, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { d as checkIsLogin, f as sendUserLogin, y as Env } from "../chunks/BwrZ3UQO.js";
import { t as Notify } from "../chunks/DwKmqXYU.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
var root = from_html(`<div class="relative"><img class="aM5 aMg" src="/images/background-1.webp" alt=""/> <div class="flex items-center h-screen aM6 relative aMg"><div class="aM8 w-full aMg"><div class="aM9 flex items-center text-xl aMg">Iniciar Sesión</div> <div class="aM7 relative mb-2 aMg"><img class="w-full h-full aMg" src="/images/genix_logo.svg" alt="Genix Logo"/></div> <div class="flex justify-center items-center text-xl font-semibold text-[#686caa] mb-4">Gestor Empresarial en la Nube para MyPes</div> <!> <!> <div class="flex w-full justify-center items-center mt-[1.4rem]"><button class="bn1 big aMa aMg"><i class="icon-login"></i> Ingresar</button></div></div></div></div>`);
function _page($$anchor, $$props) {
	push($$props, true);
	let form = proxy({
		Usuario: "",
		Password: "",
		EmpresaID: 1,
		CipherKey: ""
	});
	let isLoading = state(false);
	const sendLogin = async () => {
		if (form.Usuario.length < 4 || form.Password.length < 4) {
			Notify.failure("Debe proporcionar un usuario y una contraseña válidos");
			return;
		}
		set(isLoading, true);
		Notify.info("Enviando Credenciales...");
		const result = await sendUserLogin(form);
		console.log(result);
		set(isLoading, false);
		if (result.error) Notify.failure("Error al iniciar sesión");
	};
	onMount(() => {
		if (checkIsLogin() === 2) Env.navigate("/");
	});
	var div = root();
	var div_1 = sibling(child(div), 2);
	var div_2 = child(div_1);
	var node = sibling(child(div_2), 6);
	Input(node, {
		required: true,
		css: "mb-12 w-full text-lg",
		label: "Usuario",
		get saveOn() {
			return form;
		},
		save: "Usuario",
		type: "text"
	});
	var node_1 = sibling(node, 2);
	Input(node_1, {
		required: true,
		css: "mb-12 w-full text-lg",
		label: "Contraseña",
		get saveOn() {
			return form;
		},
		save: "Password",
		type: "password"
	});
	var div_3 = sibling(node_1, 2);
	var button = child(div_3);
	button.__click = (ev) => {
		ev.stopPropagation();
		sendLogin();
	};
	reset(div_3);
	reset(div_2);
	reset(div_1);
	reset(div);
	template_effect(() => button.disabled = get(isLoading));
	append($$anchor, div);
	pop();
}
delegate(["click"]);
export { _page as component };
