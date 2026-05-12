declare module "ngeohash" {
  export function decode(hash: string): { latitude: number; longitude: number; error: { latitude: number; longitude: number } }
  const ngeohash: {
    decode: typeof decode
  }
  export default ngeohash
}
