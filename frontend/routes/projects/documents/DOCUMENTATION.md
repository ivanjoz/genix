---
schema: 1
page_id: projects.documents
route: /projects/documents
title: Documents (Documentos)
status: implemented
visibility: tenant
description_en: >-
  The files of a project in one place: upload and delete the project's own documents, and browse the
  files attached to its stories and their comments, grouped by story.
description_es: >-
  Los archivos de un proyecto en un solo lugar: subir y eliminar los documentos propios del proyecto,
  y ver los archivos adjuntos a sus historias y a sus comentarios, agrupados por historia.
---

# Documents (Documentos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Documents (`Documentos`) gathers every file of one project (`archivos del proyecto`,
`adjuntos`): the project's own documents, such as a contract, a requirements file or a design, and a
copy of every file attached to its stories or to their comments.

Only the project's own documents are added and deleted here. A story's files are added and removed
inside the story, from **Backlog**, **Sprints** or by opening the story from this page.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **project document** (`documento del proyecto`) belongs to the project, not to any story. It is
  listed under **Project documents (Documentos del proyecto)**.
- A **story file** (`archivo de la historia`) was attached in a story's form or in one of its
  comments. It is listed under **Files in the stories (Archivos en las historias)**; a file from a
  comment is marked **In a comment (En un comentario)**.

<!-- DOC-ID: capability.upload-project-document -->
## Upload a project document (Subir un documento del proyecto)

### User intention (Intención del usuario)

Keep a file that concerns the whole project with the project.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Projects (Proyectos) → Documents (Documentos)** at `/projects/documents`.
Choose the project in the header search box ("P01 · Name"). Under **Documentos del proyecto**, use
the drop zone **Select or drop files (up to 4 MB each) (Seleccione o arrastre archivos (hasta 4 MB
cada uno))**.

### Required information and prerequisites (Requisitos previos)

The user must be on the project's team and the project must not be archived; otherwise the drop zone
is not shown.

### Business rules and rationale (Reglas y razón de negocio)

- Each file can weigh up to 4 MB, as the drop zone says, and its name up to 200 characters.
- Several files can be dropped at once; each one is uploaded and saved on its own, so a failed file
  is reported and the others continue.

### Result and side effects (Resultado y efectos)

The document appears in the list, newest first, with its size and upload date and time. Clicking it
opens or downloads the file.

### Limitations (Limitaciones)

- A file's link opens without signing in to Genix: it is hard to guess, but anyone who receives the
  link can open the file.
- Documents cannot be renamed or replaced; delete and upload again.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde subo el contrato o el documento de requerimientos del proyecto? ¿Cuál es el tamaño máximo
  de un archivo?
- Documento, archivo, adjunto, anexo, subir, cargar.

<!-- DOC-ID: capability.delete-project-document -->
## Delete a project document (Eliminar un documento del proyecto)

### Where to find it (Dónde encontrarlo)

The remove button on the document's card under **Documentos del proyecto**, then confirm "¿Eliminar
"…"? El archivo también se elimina."

### Business rules and rationale (Reglas y razón de negocio)

Any team member can delete a project document while the project is not archived.

### Result and side effects (Resultado y efectos)

The document leaves the list and the file is deleted from storage, so its link stops working.

### Limitations (Limitaciones)

Story files cannot be deleted here: "Los archivos de una historia se editan en la historia."

<!-- DOC-ID: capability.story-files -->
## Browse the files of the stories (Archivos en las historias)

### User intention (Intención del usuario)

Find a file that was attached to some story without opening the stories one by one.

### Where to find it (Dónde encontrarlo)

The **Archivos en las historias** section. Each story that has files is a box headed by its type
icon, code, title and a badge with its last sprint's name, or **Backlog** when it was never in a
sprint. Clicking the header opens the story.

### Business rules and rationale (Reglas y razón de negocio)

- Stories are listed newest first, and their files newest first, with their size.
- Files of a deleted story are not shown.

### Result and side effects (Resultado y efectos)

Saving or commenting the story opened from here refreshes the page's lists. A file removed from the
story's form is deleted from storage and leaves this page.

### Limitations (Limitaciones)

Files attached to comments cannot be removed, because comments are not edited.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde están todos los archivos adjuntos del proyecto? ¿En qué historia subieron este archivo?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Un archivo puede pesar hasta 4 MB."**: compress or split the file.
- No drop zone or remove button: the user is not on the project's team, or the project is archived.
- **"Solo el equipo del proyecto puede hacer esto."**: ask a project admin to add you to the team.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Backlog** and **Sprints** open the stories, where their files and comments are edited.
- **Projects (Proyectos)** manages the team that may upload documents.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.upload-project-document]
  - path: frontend/routes/projects/documents/+page.svelte
    role: page
    hash: sha256:84a3df1fa115e7709d7f01e6c80eeda3a51914dfa7b4e5bd267b8dd5c14f3eef
    supports: [page-purpose, capability.upload-project-document, capability.story-files]
  - path: frontend/routes/projects/documents/ProjectDocuments.svelte
    role: user-interface
    hash: sha256:e59bbf5ed083344f5cff4db9d4133e5a91487f09f736af0767a44b050d3dbb9a
    supports: [concepts, capability.upload-project-document, capability.delete-project-document, troubleshooting]
  - path: frontend/routes/projects/documents/StoryDocuments.svelte
    role: user-interface
    hash: sha256:82169b79414f8f105ec802d618abab84c48b1b740c16e3df454d5ab7b68cca40
    supports: [concepts, capability.story-files]
  - path: frontend/routes/projects/documents/documents.ts
    role: business-logic
    hash: sha256:f4ec9c1813f3eb4fd0ba767ec72066b231b1001603bcb07300e18246580a91b3
    supports: [capability.upload-project-document, capability.story-files]
  - path: frontend/routes/projects/documents/documents.svelte.ts
    role: frontend-service
    hash: sha256:a96b4795366ce07900f1f4b25df4da708b749b1bbde5528faa5181b342dc7536
    supports: [capability.upload-project-document, capability.delete-project-document]
  - path: frontend/routes/projects/story/StoryPanel.svelte
    role: user-interface
    hash: sha256:2b344d7e9dbaae834b0f101b6b6e918025222ac160cad5929e4e25aaff8d587d
    supports: [capability.story-files]
  - path: frontend/domain-components/attachments/attachments.ts
    role: shared-domain
    hash: sha256:1ea1781343d19cad145c4862073f47d1e926df92d6302268fe49aba1169b783c
    supports: [capability.upload-project-document, troubleshooting]
  - path: backend/projects/documents.go
    role: backend-handler
    hash: sha256:4abafa016bb50053a9ecf6cee725b16756171eeb783fb737db10bcf88f4844f0
    supports: [capability.upload-project-document, capability.delete-project-document]
  - path: backend/projects/attachments.go
    role: backend-handler
    hash: sha256:759052b748268f9237aa0626f71659ff2257a58f9c97dcf556636033200d90b2
    supports: [capability.upload-project-document, troubleshooting]
  - path: backend/projects/document_rules.go
    role: business-logic
    hash: sha256:8e1c88479c91031569dfa97e914bff1f9f1fb25869393efb80901be26c20ed7f
    supports: [concepts, capability.story-files]
  - path: backend/projects/stories.go
    role: backend-handler
    hash: sha256:9b6f2cf5a86a59be861fd8768a2ac699ba783504a2d08f9664e0322d2177b78f
    supports: [capability.story-files]
  - path: backend/projects/project_access.go
    role: business-logic
    hash: sha256:7dad83cea78e4d9da0d9d386b5cea012143a095ca13a91153cd07aaf16db2659
    supports: [capability.upload-project-document, capability.delete-project-document, troubleshooting]
  - path: backend/projects/types/documents.go
    role: data-model
    hash: sha256:3ea6e50d9695e43c20937cdbb08fa0f166f2bc9b18e6936439f8105378206447
    supports: [concepts]
  - path: backend/projects/types/stories.go
    role: data-model
    hash: sha256:33d0fca372da7917d1abc4a46ba6410033bc4e4ca1f79d6cc487179617afbe8a
    supports: [capability.upload-project-document, capability.story-files]
```
