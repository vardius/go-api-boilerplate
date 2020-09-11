import {extendTheme} from "@chakra-ui/core"

const theme = extendTheme({
  colors: {
    brand: {
      light: {
        background: "#323031",
        background20: "#3D3B3C",
        primary: "#7F7979",
        secondary: "#C1BDB3",
        active: "#5F5B6B",
      },
      dark: {
        background: "#2E5266",
        background20: "#6E8898",
        primary: "#9FB1BC",
        secondary: "#D3D0CB",
        active: "#E2C044",
      },
    },
  }
});

export default theme;
