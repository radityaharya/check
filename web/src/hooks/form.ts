import { createFormHook } from '@tanstack/react-form';
import {
  TextField,
  NumberField,
  SelectField,
  TextAreaField,
  CheckboxField,
  TagsField,
  SubmitButton,
} from '../components/ui/form-fields';
import { fieldContext, formContext } from './form-context';

export const { useAppForm } = createFormHook({
  fieldComponents: {
    TextField,
    NumberField,
    SelectField,
    TextAreaField,
    CheckboxField,
    TagsField,
  },
  formComponents: {
    SubmitButton,
  },
  fieldContext,
  formContext,
});
