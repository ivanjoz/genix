El objeto es poder generar secciones de UI reutilizables y editables que el usuario pueda seleccionar y guardar para que pueda crear su propio ecommerce.

Esta biblioteca de secciones permitirán al usuario tener plantillas de secciones prediseñadas.

Las secciones de UI reutilizables serán del tipo ComponentAST y se renderizará mediante frontend/pkg-store/renderer/EcommerceRenderer.svelte.

Cada sección podrá tener variables del tipo ComponentASTVariable, que son valores dinámicos que se evaluarán y se generarán las clases de tailwind correspondientes.

Los tipos de variables son en sí los prefijos de tailwind como: w, h, p, mb, mt, etc.

Cada componente debe estar en un archivo .ts con una pequeña descripción de cómo se ve. Por ejemplo: Sección con fondo color solido, texto lateral derecho, imagen lateral izquierda, botón de "buscar productos". Esa descripción servirá para luego crear un una imagen representativa del layout del componente para una mejor visualización en la biblioteca de componentes.

El componente frontend/pkg-store/renderer/EcommerceRenderer.svelte tomará la lista de secciones y las renderizará y creará una página ecommerce.

También existirán componentes personalizados como ProductCard o ProductCardHorizonal que podrán ser utilizados mediante el tagName, si el tagName en vez de ser un DIV o BUTTON o un componente HTML, es un componente personalizado, entonces renderiza ese componente.

funcionalidad de componentes con ITextLine[]. Las lineas de texto permiten al usuario darle un estilo distinto a cada linea. Se necesitará un componente que tome esos valores ITextLine[] y los convierta en un formulario editable, donde el usuario puede agregar más bloques de texto y editar el estilo de cada bloque de texto, algo básico como tamaño, color de texto, alineación.

Los componentes también pueden usar colores globales. Los colores globales son una paleta de colores de 10 colores desde el más oscuro hasta el más claro. Las variables para usar estos colores globales son __COLOR:1__ y serán reemplazas madiante un regex. Se asume que los 10 colores ya están pre seteados por el usuario.
