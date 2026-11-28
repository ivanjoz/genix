---
schema: 1
page_id: hr.departments-positions
route: /hr/departments-positions
title: Departments & Positions (Áreas y Cargos)
status: implemented
visibility: tenant
description_en: >-
  Define the company's departments, with the department each reports to, its manager and its cost
  function (administration, sales or production), and the positions inside each department with
  their SUNAT occupation code. A second tab draws the org chart of the active workers from each one's
  immediate boss.
description_es: >-
  Define las áreas de la empresa, con el área de la que dependen, su responsable y su función de
  costo (administración, ventas o producción), y los cargos de cada área con su código de ocupación
  SUNAT. Otra pestaña dibuja el organigrama de los trabajadores activos según el jefe inmediato de
  cada uno.
---

# Departments & Positions (Áreas y Cargos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Departments & Positions (`Áreas y Cargos`) defines the structure workers are placed in: the
company's departments (`áreas`) and the positions (`cargos`, `puestos`) inside each one. It also
shows the org chart (`organigrama`) of the active workers.

The page does not register workers or assign them to a department, position or boss: that is done
in Employees & Contracts (`Empleados & Contratos`). The org chart only reflects those assignments.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **department** (`área`) can report to another department (**Reports to (Depende de)**) and have
  a **Manager (Responsable)**, an active worker.
- The **cost function** (`función de costo`) of a department is **Administration (Administración)**,
  **Sales (Ventas)** or **Production (Producción)**: the function its workers' payroll cost belongs
  to. Each payslip records the cost function of the worker's department.
- A **position** (`cargo`) belongs to exactly one department and can carry its **SUNAT occupation
  code (Código de ocupación SUNAT)**, used for T-Registro.

<!-- DOC-ID: capability.manage-departments -->
## Create, edit or delete departments (Áreas)

### User intention (Intención del usuario)

Set up the company's departments before registering workers, or reorganize them.

### Where to find it (Dónde encontrarlo)

**Company (Empresa) → HR (RRHH) → Departments & Positions (Áreas y Cargos)** at
`/hr/departments-positions`, tab **Departments & Positions (Áreas y Cargos)**. In the **Departments
(Áreas)** table, the green **+** button opens **New department (Nueva área)**; clicking a row opens
**Edit department (Editar área)**, which can also delete it. Columns: **Department (Área)**,
**Reports to (Depende de)**, **Cost function (Función de costo)** and **Staff (Personal)**, the
count of active workers.

### Required information and prerequisites (Requisitos previos)

**Name (Nombre)** and **Cost function (Función de costo)** (Administration by default) are
required; **Reports to (Depende de)** and **Manager (Responsable)** are optional.

### Business rules and rationale (Reglas y razón de negocio)

- The name needs at least 2 characters.
- A department cannot report to itself, and the parent department must not be deleted.
- A department cannot be deleted while it still has active workers, active positions or departments
  reporting to it; the message names the first one found so it can be moved first.

### Result and side effects (Resultado y efectos)

The department is offered when registering workers and positions. Payslips calculated afterwards
carry its cost function. Hours vs payroll lists the Planilla workers of Production departments even
when they logged no hours.

### Limitations (Limitaciones)

**Reports to** and **Manager** do not draw the org chart; the chart follows each worker's immediate
boss.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un área? ¿Por qué no puedo eliminar un área? → it still has staff, positions or child
  departments.
- ¿Qué es la función de costo? Área, departamento, gerencia, sección, centro de costo.

<!-- DOC-ID: capability.manage-positions -->
## Create, edit or delete positions (Cargos)

### User intention (Intención del usuario)

Define the job titles workers can hold in each department.

### Where to find it (Dónde encontrarlo)

Same tab, **Positions (Cargos)** table: the green **+** button opens **New position (Nuevo cargo)**;
clicking a row opens **Edit position (Editar cargo)**, which can also delete it. Columns: **Position
(Cargo)** and **Department (Área)**.

### Required information and prerequisites (Requisitos previos)

**Name (Nombre)** and **Department (Área)** are required; **SUNAT occupation code (Código de
ocupación SUNAT)** is optional. The department must exist first.

### Business rules and rationale (Reglas y razón de negocio)

- The name needs at least 2 characters and the department must not be deleted.
- A position cannot be deleted while an active worker holds it; the message names the worker.

### Result and side effects (Resultado y efectos)

In Employees & Contracts the **Position (Cargo)** field only lists the positions of the worker's
department.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo agrego un cargo? ¿Por qué no aparece el cargo al registrar un trabajador? → it belongs to
  another department.
- Cargo, puesto, ocupación, función.

<!-- DOC-ID: capability.org-chart -->
## See the org chart (Organigrama)

### User intention (Intención del usuario)

See who reports to whom across the company.

### Where to find it (Dónde encontrarlo)

Tab **Org chart (Organigrama)**.

### Business rules and rationale (Reglas y razón de negocio)

- Each card shows an active worker's name and position, nested under his **Immediate boss (Jefe
  inmediato)** from Employees & Contracts, sorted by name.
- The department name heads a card only where it changes from the boss's department.
- A worker without an active boss is drawn at the top. Terminated and deleted workers are not drawn.

### Limitations (Limitaciones)

The chart is read only; change a boss in Employees & Contracts. Without workers it says there are
no employees yet.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo veo el organigrama? ¿Por qué un trabajador aparece arriba sin jefe? → his immediate boss is
  empty or terminated.
- Organigrama, estructura organizacional, jerarquía.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"el área tiene personal activo (…)":** move those workers to another department first.
- **"el área tiene el cargo "…"; elimínelo o muévalo antes":** delete or move the position.
- **"el área "…" depende de ella; muévala antes":** change the child department's **Reports to**.
- **"El cargo está asignado a …; reasígnelo antes de eliminarlo":** change that worker's position.
- **"Seleccione la función de costo del área…":** choose Administration, Sales or Production.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Employees & Contracts (Empleados & Contratos):** assigns workers to departments, positions and
  bosses.
- **Payroll (Planilla):** payslips show the worker's department.
- **Hours vs payroll (Horas vs planilla):** uses the Production cost function.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.manage-departments]
  - path: frontend/routes/hr/departments-positions/+page.svelte
    role: page
    hash: sha256:79ac521f5d6c47c24552b77ee12ab1ddc1d1bc18d12853813c395cd8418947be
    supports: [page-purpose, capability.manage-departments, capability.org-chart]
  - path: frontend/routes/hr/departments-positions/OrganizationTab.svelte
    role: user-interface
    hash: sha256:fa18bf0ac5549780879bfec88c6b917cfdf56a108551ad75897348eb60ceb766
    supports: [concepts, capability.manage-departments, capability.manage-positions]
  - path: frontend/routes/hr/departments-positions/OrgChartTab.svelte
    role: user-interface
    hash: sha256:a6c69f6fcf533aeb75d1dda9e19a7d5d9eac0c4310931fab60cdc6a765c3cecb
    supports: [capability.org-chart]
  - path: frontend/routes/hr/employees/employees.ts
    role: business-logic
    hash: sha256:590c1af5be318be15985ecee8eba31897d03d23728093762326e929cf1748672
    supports: [capability.manage-departments, capability.org-chart]
  - path: frontend/routes/hr/employees/EmployeesTab.svelte
    role: user-interface
    hash: sha256:144a74c35afe8c39abed7fbc6be202e8f3b7d25d665a8505982407c648169a5b
    supports: [capability.manage-positions]
  - path: frontend/services/hr/organization.svelte.ts
    role: frontend-service
    hash: sha256:53cd040e67573ebf1d4ed8a53efbcee72a1d078c392b724ff1d63744d304ea69
    supports: [capability.manage-departments, capability.manage-positions]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts]
  - path: backend/hr/organization.go
    role: backend-handler
    hash: sha256:8af3a97fbf7a65a08063175906848d81f3a1540dc50866132c2a5a5590baeb87
    supports: [capability.manage-departments, capability.manage-positions, troubleshooting]
  - path: backend/hr/types/organization.go
    role: data-model
    hash: sha256:7f3af952a83ac3d23380f410601127e89c01aaaaaeb1e1e03395d183ac3b0fe0
    supports: [concepts]
  - path: backend/hr/types/catalogs.go
    role: data-model
    hash: sha256:78e2bcef74a7aaeba64dee514ec8a16f5523c38ac2f51bac8fec5fde8dcf9d70
    supports: [concepts]
  - path: backend/hr/payroll_calculate.go
    role: business-logic
    hash: sha256:0cbda12fb8d81d3c6582ca08f9011fb3af283dc22c059653eaf84e24048009b2
    supports: [concepts, capability.manage-departments]
  - path: frontend/routes/production/labor-hours/labor-hours.ts
    role: shared-domain
    hash: sha256:4ae9afe3dfae8ece1fc278184ac7e87597a4c917c2fed53ecfba3f9d244826f7
    supports: [capability.manage-departments, related-pages]
```
