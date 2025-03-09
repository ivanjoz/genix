import { GetSignal, makeApiGetHandler } from "~/shared/http"

export interface IGaleriaImagen {
  Image: string
	Description: string
	ss:      number
	upd:     number
}

export const useGaleriaImagesAPI = () => makeApiGetHandler<IGaleriaImagen[]>(
  { route: "galeria-images", emptyValue: [],
    errorMessage: 'Hubo un error al obtener las sedes / almacenes.',
    cacheSyncTime: 1, mergeRequest: true,
    useIndexDBCache: 'galeria_images',
  },
  (result_) => {
    const images = result_.Records as IGaleriaImagen[]
    images.sort((a,b) => b.upd - a.upd)
   // const result = result_ as IGaleriaImagen[]
    return images
  }
)
