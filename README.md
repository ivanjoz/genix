# Genix: ERP + Ecommerce para pequeños negocios

Stack:
Frontend: Svelte.js
Backend: Go
Base de Datos: Scylla / Apache Cassandra

La documentación detallada está en:
Frontend: frontend2/README.md

## Migración de frontend desde Solid.js
Actualmente estamos migrando el frontend de solid.js (/frontend) a svelte.js (/fontend2) para poder continuar con los siguientes puntos de nuestro roadmap:

- Compilación en el cliente de componentes UI de Ecommerce
- Editor WYSIWYG de Ecommerce.

Importante: El frontend en solid.js no usa Tailwind sino clases css similares en global.css, convertirlas a Tailwind al migrar.
Hay clases que sí están en Solid.js como: ff-bold, ff-mono, ff-semibold, c-red, c-blue y demás colores que comienzan con c*.

La presente documentación está en curso.
