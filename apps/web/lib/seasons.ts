export type SeasonPalette = {
  accent: string;
  heading: string;
  body: string;
  button: string;
  buttonShadow: string;
};

export type Season = {
  name: string;
  label: string;
  image: string;
  palette: SeasonPalette;
};

export const seasons: Season[] = [
  {
    name: "Spring",
    label: "MAR · APR · MAY",
    image: "/spring1.jpg",
    palette: {
      accent: "#F4B6C2",
      heading: "#F5F1E8",
      body: "#D6E3D1",
      button: "#F4B6C2",
      buttonShadow: "rgba(244,182,194,0.24)",
    },
  },
  {
    name: "Summer",
    label: "JUN · JUL · AUG",
    image: "/summer.jpg",
    palette: {
      accent: "#FFD580",
      heading: "#FFF8E8",
      body: "#E8DCC4",
      button: "#F5C36E",
      buttonShadow: "rgba(245,195,110,0.24)",
    },
  },
  {
    name: "Autumn",
    label: "SEP · OCT · NOV",
    image: "/autumn.jpg",
    palette: {
      accent: "#E8A87C",
      heading: "#F4E8D8",
      body: "#E0CDB8",
      button: "#C77E4A",
      buttonShadow: "rgba(199,126,74,0.26)",
    },
  },
  {
    name: "Winter",
    label: "DEC · JAN · FEB",
    image: "/winter.jpg",
    palette: {
      accent: "#B8D4E8",
      heading: "#F0F5FA",
      body: "#D8DFE6",
      button: "#8AA8C5",
      buttonShadow: "rgba(138,168,197,0.24)",
    },
  },
];

export const COLOR_TRANSITION =
  "color 600ms ease, background-color 600ms ease, border-color 600ms ease, box-shadow 600ms ease";
