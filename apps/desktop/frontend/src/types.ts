export type KVPair = {
  id: string
  name: string
  value: string
  enabled: boolean
}

export type BodyType = 'none' | 'json' | 'raw' | 'form-urlencoded' | 'multipart'

export type BodyState = {
  type: BodyType
  content: string
}

export type AuthType = 'none' | 'bearer' | 'basic' | 'apikey'

export type AuthState =
  | { type: 'none' }
  | { type: 'bearer'; token: string }
  | { type: 'basic'; username: string; password: string }
  | { type: 'apikey'; key: string; value: string; location: 'header' | 'query' }
