/**
 * Context key marking that the AST is being rendered inside the store builder
 * (editable preview) rather than on a live storefront. Components read it to opt into
 * builder-only behaviour (e.g. EcommerceSlider syncs its slide with the editor and
 * disables autoplay). Set once by BuilderSectionRender; absent → production rendering.
 */
export const EC_BUILDER_MODE = Symbol('ec-builder-mode');
