// neo-blessed is API-compatible with blessed
// Use blessed type definitions
declare module "neo-blessed" {
  export * from "blessed";
  import blessed from "blessed";
  export default blessed;
}
