import React from 'react'
import {
  BooleanInput,
  Edit,
  PasswordInput,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'

const OpenListTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.openlist.name', {
    smart_count: 1,
  })
  return <Title subTitle={resourceName} />
}

const OpenListEdit = (props) => {
  return (
    <Edit id="openlist" title={<OpenListTitle />} actions={false} {...props}>
      <SimpleForm variant={'outlined'}>
        <BooleanInput source="enabled" />
        <TextInput source="openlistBase" fullWidth />
        <TextInput source="openlistUser" />
        <PasswordInput
          source="openlistPass"
          helperText="resources.openlist.message.keepPassword"
        />
        <BooleanInput source="coverEnabled" />
        <BooleanInput source="streamEnabled" />
      </SimpleForm>
    </Edit>
  )
}

export default OpenListEdit
