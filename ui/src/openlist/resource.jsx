import React from 'react'
import { Resource } from 'react-admin'
import openlist from './index'

export const renderOpenListResource = (permissions) => {
  if (permissions !== 'admin') {
    return null
  }

  return (
    <Resource name="openlist" {...openlist} options={{ subMenu: 'settings' }} />
  )
}
