import { useEffect } from 'react'
import { useRedirect } from 'react-admin'

const OpenListList = () => {
  const redirect = useRedirect()

  useEffect(() => {
    redirect('/openlist/openlist')
  }, [redirect])

  return null
}

export default OpenListList
