export interface IInput {
  id?: number;
  saveOn: any;
  save: string;
  label?: string;
  css?: string;
  inputCss?: string;
  required?: boolean;
  validator?: (v: string | number) => boolean;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  onChange?: () => void;
  postValue?: any;
  baseDecimals?: number;
  content?: string | any;
  transform?: (v: string | number) => string | number;
  useTextArea?: boolean;
  rows?: number;
}
