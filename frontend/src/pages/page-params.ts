import { IPageParams } from "./page";

export const pageParams: IPageParams[] = [
  { id: 1, name: "Sección Básica",
    params:   [
      { key: 'title', name: 'Título', type: 1 },
      { key: 'subtitle', name: 'Subtítulo', type: 1 },
      { key: 'content', name: 'Contenido', type: 2 },
    ]
  },
  { id: 10, name: "Header de Página",
    params:   [
      { key: 'title', name: 'Título', type: 1 },
      { key: 'subtitle', name: 'Subtítulo', type: 1 },
    ]
  },
  { id: 21, name: "Capa con imagen de fondo",
    params:   [
      { key: 'title', name: 'Título', type: 1 },
      { key: 'content', name: 'Contenido', type: 1 },
    ]
  },
]