export interface IOption {
	i: number, v: string
}

export const productoMonedaOptions: IOption[] = [
  { i: 1, v: "PEN (S/.)" },
  { i: 2, v: "USD ($)" },
]

export const productoUnidadOptions: IOption[] = [
  { i: 1, v: "Kg" },
  { i: 2, v: "g" },
  { i: 3, v: "Libras" },
]
