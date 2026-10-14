// The configuration of the mygo CLI, which builds the desktop app
// (see https://github.com/egoist/mygo/blob/main/docs/configuration.md).
// build.sh sets NEW_API_VERSION from the git tag.
export default {
  name: "New API",
  identifier: "com.newapi.desktop",
  version: process.env.NEW_API_VERSION || "0.0.0",
  copyright: "© QuantumNous",
  out: "build",
  linux: {
    comment: "AI Model Gateway",
    categories: ["Development"],
  },
};
