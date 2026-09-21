type SessionSortState = { id:string; dir:'asc'|'desc' } | null

declare global {
  interface Object {
    sort?: SessionSortState
  }
}

export {}
