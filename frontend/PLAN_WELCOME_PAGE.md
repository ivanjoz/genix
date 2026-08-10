# Welcome page redesign plan

Replace the current photo-and-card `/login` screen with a bilingual, responsive `/welcome` landing page that explains Genix, preserves sign-in, shows an honest product roadmap, offers registration, and ends with a contact form. The page will be public and chrome-less, use the existing Genix UI runtime and `T.svelte` translation convention, and make `/welcome` the canonical authentication entry point.

## 1. Confirmed scope and remaining decisions

The registration scope is now confirmed; only the unrelated routing/contact decisions remain open:

1. **Registration form — confirmed: email only.** The modal collects one value: the user's email address. It does not collect company name, RUC, user name, password, or verification code.
2. **Registration endpoint — confirmed: deferred.** The public endpoint that sends the registration email will be implemented in another stage. The current landing-page stage builds the modal, explanatory copy, validation, and a clean integration boundary, but must not fake a successful request or create Company/User records.
3. **Route replacement — recommended: no legacy alias.** Move the public entry point from `/login` to `/welcome`, update every internal auth redirect, and remove the old route in line with the project's pre-alpha/no-backwards-compatibility policy. If external bookmarks must keep working, explicitly approve a temporary redirect before implementation.
4. **Contact delivery and public details.** Confirm how the separate contact form will submit, its destination email, whether a phone/WhatsApp link should appear, and whether the GitHub repository should be linked in the footer.

## 2. Current-state findings

- `frontend/routes/login/+page.svelte` is the only authentication page. It is a full-viewport stock image with a fixed white login card, username/password fields, an environment selector, and sign-in logic.
- No `frontend/routes/welcome/` route exists on this branch. The deployed `/welcome` URL currently receives the static SPA shell, but the local application has no matching page component.
- `/login` is hard-coded as public and/or layout-free in several places:
  - `frontend/routes/+layout.svelte`
  - `frontend/libs/ui-runtime.svelte.ts`
  - `frontend/domain-components/Page.svelte`
  - `frontend/domain-components/HeaderConfig.svelte`
- The current login service in `frontend/services/login.ts` can be reused without changing its credential exchange. Successful login already routes to initial setup or the authenticated home page.
- `T.svelte` translates `English|Spanish` strings reactively through the injected UI runtime. `Core.languaje` and `setLanguaje` already provide a persisted Spanish/English selector, but the current public login screen does not expose one.
- The shared `Modal.svelte` supports responsive sizing and translated save/close actions. It currently lacks `role="dialog"`, `aria-modal`, labelled-dialog wiring, Escape handling, focus trapping, and focus restoration; these should be fixed in the shared component rather than worked around only on `/welcome`.
- The authenticated `POST.company` and `POST.usuarios` handlers must not be reused for public sign-up. The public registration-email endpoint and the later verified registration flow are explicitly deferred to another stage.
- The repository README is the authoritative source for product capabilities and roadmap status. Marketing copy must preserve the stated **pre-alpha** status and must not claim unfinished features are available.

## 3. Information architecture and page flow

### 3.1 Glass navigation

Create a sticky top navigation inside the public page rather than using the authenticated `AppHeader`.

- Left: Genix logo linked to the hero.
- Center on desktop: `Inicio`, `Funcionalidades`, `Roadmap`, and `Contacto` anchor links.
- Right: compact `ES / EN` language control, secondary `Ingresar` action that scrolls to and focuses the login card, and primary `Regístrese` action that opens the registration modal.
- Visual treatment: translucent white/indigo surface, `backdrop-blur`, subtle light border, soft shadow, rounded container, and a solid fallback background for browsers without backdrop filtering.
- Mobile: logo, language toggle, and menu button remain visible; section links and actions open in an accessible disclosure panel. Close it after navigation and on Escape.

### 3.2 Hero and login

Use a two-column hero on desktop and a single-column flow on mobile.

- Content column:
  - Eyebrow: open-source ERP + e-commerce for small businesses.
  - Clear headline focused on operating the business from one system.
  - Short description covering sales, inventory, purchasing, finance, and online sales without overloading the first screen.
  - Primary `Regístrese` and secondary `Conozca Genix` calls to action.
  - Small trust row: open source, self-hostable, tenant data export.
- Login column:
  - Keep username, password, server/environment selector, loading state, validation, and the existing `sendUserLogin` behavior.
  - Add a concise heading and supporting sentence; do not repeat the complete product pitch inside the form.
  - Submit on form submit/Enter, disable the button while pending, and preserve the current notification behavior.
- Media treatment: reuse `/images/background-1.webp` only as a cropped, layered hero accent with an indigo overlay rather than as the full-page background. Keep the rest of the page product-focused with gradients, restrained decorative shapes, and the existing logo assets.

### 3.3 Product explanation

Add a short “What is Genix?” section with three differentiators grounded in the README:

- One ERP and e-commerce workspace for small businesses.
- Flexible cloud, self-hosted, and single-binary deployment.
- Data ownership, tenant isolation, and full backup/export.

Avoid deep technical terms such as Scylla partition keys or serializer internals in the primary copy. Those belong in an optional technical/open-source link, not in the sales narrative.

### 3.4 Feature groups

Use six consistent cards with icons, short descriptions, and no unsupported claims:

1. **Products and customers:** catalogs, categories, brands, attributes, images, customers, suppliers, and Excel import/export.
2. **Inventory and purchasing:** stock by warehouse, lots/serials, movements, purchase orders, goods receipt, supplies, and replenishment planning.
3. **Sales and point of sale:** order creation, payments, delivery status, reports, charts, and sales planning.
4. **Cash and finance:** cash/bank registers, income and expenses, movements, reconciliation, recurring expenses, and partial payments.
5. **E-commerce:** visual storefront builder, product search/catalog, cart/payment UI, shipping configuration, and custom-domain deployment. Mark checkout order persistence and customer accounts as planned, not complete.
6. **AI, security, and operations:** AI-assisted webpage building, in-app ERP navigation, users/access profiles, automated jobs, and tenant backups.

Each card should have a semantic heading and readable copy rather than relying on icons alone.

### 3.5 Roadmap

Show a distilled, customer-readable roadmap rather than copying the entire technical README.

- **Available now:** core ERP workflows, product/customer management, warehouse and purchasing, POS/sales, cash management, website builder, access control, and backups.
- **In progress:** vendor-free/self-hosted provider completion, weekly sales summary, projected cash-flow report, and shipping-price checkout integration.
- **Next:** e-commerce order persistence, storefront customer accounts/order history, streaming AI responses, and token quota enforcement.

Use explicit status labels (`Disponible`, `En progreso`, `Próximo`) plus text/icons; do not encode status by color alone. Add a visible pre-alpha note explaining that priorities and interfaces can change. Keep `README.md` as the source of truth and review the landing-page summary whenever its roadmap changes.

### 3.6 Contact section and footer

Place the contact form near the end on a contrasting surface.

- Proposed fields: name, email, company (optional), subject (optional), and message.
- Required fields: name, email, and message.
- Include clear pending, success, and failure states; on success, clear the fields and leave a persistent confirmation message.
- Add a honeypot field that is visually hidden but available to the backend's spam check.
- Footer: compact logo/description, navigation links, GPL v3/open-source note, and confirmed public contact/GitHub links.

## 4. Registration modal

The `Regístrese` buttons in the header and hero must open the same modal ID through the shared UI runtime.

### Field

| Field | Requirement | Validation |
|---|---|---|
| Email | Required | Trimmed, lower-cased, valid plain email address, max 254 characters |

Email must be the modal's only input. The verification code belongs to the future registration continuation screen, not to this initial form.

### Explanatory copy

Place a short explanation immediately above the email field. The meaning must remain equivalent to:

- `Enter your email address to begin registration.|Ingrese su correo electrónico para iniciar el registro.`
- `We will send you a registration email. Click the registration link in that email, or copy its verification code, to continue.|Le enviaremos un correo de registro. Para continuar, haga clic en el enlace de registro incluido en el correo o copie su código de verificación.`

The primary action should be labelled `Send registration email|Enviar correo de registro` once the public endpoint is available.

### Behavior

- Use the shared `Modal.svelte` and shared `Input.svelte` components with the 24-column, `gap-10` form grid.
- Use a title snippet containing `T.svelte`, and translate all labels, help text, actions, notifications, validation messages, and ARIA labels.
- Explain directly in the modal that the user must either click the registration link or copy the verification code from the email to continue.
- In this frontend-only stage, do not show a fake success state. Keep endpoint invocation isolated behind a future `requestRegistrationEmail(email)` function and wire it only when the public endpoint exists.
- When the endpoint stage is implemented, disable duplicate submission while pending, preserve the email after failure, and replace the form with a confirmation state after success.
- The future confirmation state should repeat where to find the link/code and offer a way to correct the email address. Resend limits and the code-entry continuation screen belong to the endpoint/registration stage.
- Close on explicit close and Escape, and return focus to the button that opened it.
- Do not collect company data, a user name, a password, or a verification code in this modal.

## 5. Translation rules

- Render every static visible string through `T.svelte` using `English|Spanish` source text.
- Use `tr(...)` only where a component/API requires a plain string, such as notifications, `aria-label`, document metadata chosen at runtime, and validation errors.
- Add a public language selector using `setLanguaje(1 | 2)` so the page's bilingual content is user-controllable and persists through the existing storage mechanism.
- Translate navigation labels, form labels, validation, success/error feedback, modal content, feature descriptions, roadmap statuses, legal/pre-alpha copy, image alt text, and SEO title/description.
- Keep product names and legal identifiers such as `Genix` and `GPL v3` unchanged.

## 6. Proposed file changes

### Frontend

- Add `frontend/routes/welcome/+page.svelte`: public page layout, section data, hero login, navigation, language control, anchor behavior, and contact form.
- Add `frontend/routes/welcome/RegistrationModal.svelte`: the email-only registration form, translated explanation, client validation, and the future submission boundary.
- Add `frontend/routes/welcome/welcome.md`: a 2–4 line route description for agent navigation, per frontend conventions.
- Remove `frontend/routes/login/+page.svelte` after its login behavior is migrated.
- Update `frontend/routes/+layout.svelte` to make `/welcome` public and layout-free, skip access-catalog loading there, and redirect unauthenticated users there.
- Update `frontend/libs/ui-runtime.svelte.ts`, `frontend/domain-components/Page.svelte`, and `frontend/domain-components/HeaderConfig.svelte` so logout/session expiry/auth guards consistently use `/welcome`.
- Update public-route examples in `frontend/packages/genix-ui/SECURITY.md`, `frontend/packages/genix-ui/README.md`, and the `/login` reference in `frontend/docs/UI_COMPONENTS.md` so documentation matches runtime behavior.
- Harden `frontend/packages/genix-ui/layers/Modal.svelte` with dialog semantics, keyboard behavior, initial focus, focus trapping, focus restoration, and scroll locking. Preserve its existing API unless a minimal optional `initialFocus` prop is necessary.

### Deferred registration endpoint stage (not part of this landing-page stage)

- Add a small typed `requestRegistrationEmail(email)` frontend service only when the endpoint contract exists; do not add an unused service now.
- Add a deliberately public endpoint that accepts only the normalized email plus anti-abuse metadata. It must send a registration email containing both a signed, single-use registration link and a short-lived verification code.
- The link and code must continue into the same verified registration flow. The continuation route/screen, company details, main-user details, password setup, expiration rules, and resend behavior are all part of that later stage.
- Validate email again on the server, return a generic accepted response that does not reveal whether the address already exists, and apply per-IP and per-email throttling.
- Add honeypot/bot protection and small request limits before exposing the endpoint.
- Do not log the raw body or email address. Log only request ID, outcome, and non-PII diagnostic information.
- Do not create a Company/User record until the email link or code has been verified and the later registration form has passed server-side validation.

## 7. Visual and responsive specification

- Use Tailwind v4 utilities and remember this project's `--spacing: 1px`; spacing values such as `p-24` mean 24 px, not Tailwind's default scale.
- Retain the Genix indigo/violet identity, with a neutral white/slate content background and one warm accent for calls to action/status. Maintain WCAG AA text contrast on glass and image surfaces.
- Use `font-semibold`, `font-bold`, and `text-*` Tailwind utilities; do not set font weight or font size in custom CSS.
- Keep custom CSS limited to effects Tailwind cannot express cleanly, such as a specific background composition. Respect `prefers-reduced-motion` and avoid scroll-linked/parallax animation.
- Desktop: centered max-width content, two-column hero, six cards in a three-column grid, and horizontal roadmap columns.
- Tablet: two-column feature grid and stacked/condensed hero as space requires.
- Mobile: one-column content, login card after hero copy, full-width actions, compact glass header, disclosure menu, stacked roadmap, and nearly full-width registration modal.
- Add section scroll margins so sticky navigation does not cover anchored headings.

## 8. Accessibility, semantics, and agent navigation

- Use semantic `header`, `nav`, `main`, `section`, `form`, and `footer` landmarks with one page-level `h1` and ordered headings.
- Give the nav, login form, registration form, and contact form descriptive `aria-label` values in the active language.
- Ensure every input has a real label, required state, associated validation message, and appropriate autocomplete value.
- Ensure all interactions work with keyboard only and show visible focus rings.
- Add `aria-current` to the active navigation item if scroll-spy behavior is included; otherwise keep the navigation simple and avoid unnecessary scroll observers.
- Provide useful alt text for meaningful media and empty alt text for decorative shapes.
- Maintain the Genix agent-navigation contract on actionable buttons through descriptive labels.

## 9. SEO and public-page behavior

- Set a route-specific title and concise bilingual description in `svelte:head`.
- Add canonical `/welcome` metadata and appropriate Open Graph fields once the production hostname and social image are confirmed.
- Avoid indexing login-specific form values or embedding environment configuration in metadata.
- If the root `/` behavior is later changed for logged-out visitors, handle that as a separate routing decision; this plan only replaces the auth entry route.

## 10. Implementation sequence

1. Confirm the remaining route/contact decisions in section 1 and finalize bilingual marketing/contact copy.
2. Create the `/welcome` route and move the existing login state/behavior into the hero card without changing the backend login contract.
3. Build the glass navigation, language selector, content sections, roadmap, contact form, and responsive styling.
4. Build the email-only registration modal, its link-or-code explanation, and harden the shared modal's accessibility behavior.
5. Replace hard-coded `/login` references and remove the old route.
6. Add focused frontend tests and run the frontend validation suite.
7. Review the result at mobile, tablet, desktop, keyboard-only, Spanish, and English breakpoints before deployment.
8. In the later registration stage, implement the public email endpoint, email link/code generation, verified continuation flow, abuse controls, and typed frontend service.

## 11. Validation plan

### Frontend checks

- Run `bun run check` from `frontend/`.
- Run `bun run build:main` from `frontend/`.
- Run `bun run scripts/analyze-dag.ts` from `frontend/` after adding the route/component imports.
- Verify `/welcome` loads while logged out, does not load the authenticated shell/access catalog, and redirects an already authenticated user to the normal application entry if that remains desired.
- Verify logout, session expiry, and unauthorized page access all navigate to `/welcome`.
- Verify sign-in success, sign-in failure, Enter submission, loading/disabled state, and server selector behavior.
- Verify both registration triggers open one modal containing exactly one input, reject invalid/empty email, and show the translated link-or-code continuation explanation.
- Verify this stage does not fake a successful registration request while the public endpoint is absent.
- Verify contact validation, honeypot behavior, failure retention, success reset, and persistent feedback.
- Verify every visible string in Spanish and English, including modal/action text and errors.
- Test keyboard navigation, focus trap/restoration, Escape close, screen-reader dialog naming, focus visibility, contrast, reduced motion, and 320 px width without horizontal overflow.

### Deferred registration-stage checks

- Add tests for email validation, generic accepted responses, honeypot rejection, per-IP/per-email throttling, link/code expiration, one-time use, and maximum payload size.
- Confirm the link and manually copied code resolve to the same verified continuation state.
- Confirm public requests do not require an auth token and cannot directly reach Company/User creation paths.
- Confirm logs never contain the email, verification code/token, or raw public-form body.
- Run targeted Go tests for the new package/handler, then `go test ./...` from `backend/`.

## 12. Acceptance criteria

- `/welcome` is the sole canonical public sign-in route and all internal redirects use it.
- The page contains a glass navigation menu, responsive hero/login, project explanation, feature overview, roadmap, registration action, contact form, and footer.
- The existing login contract and environment selector continue working.
- `Regístrese` opens a translated modal whose only input is the email address.
- The modal clearly says that continuing registration requires clicking the link in the registration email or copying its verification code.
- The current landing-page stage does not call or simulate the deferred public registration endpoint.
- All public copy and form feedback switch reactively between Spanish and English through the existing translation runtime, with static markup rendered by `T.svelte`.
- Roadmap claims match the README and clearly distinguish available, in-progress, and planned work.
- Forms are accessible, keyboard-operable, validated on both client and server when submission is enabled, and protected against obvious automated abuse.
- The page is usable at mobile, tablet, and desktop sizes, passes the project checks/build, and introduces no dependency-layer violation.
