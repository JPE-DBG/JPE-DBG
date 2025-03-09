const FONT_FAMILY = '"Source Sans 3", Arial';

const FONT = {
  light: {
    fontFamily: FONT_FAMILY,
    fontWeight: 300,
  },
  regular: {
    fontFamily: FONT_FAMILY,
    fontWeight: 400,
  },
  medium: {
    fontFamily: FONT_FAMILY,
    fontWeight: 500,
  },
  semiBold: {
    fontFamily: FONT_FAMILY,
    fontWeight: 600,
  },
  bold: {
    fontFamily: FONT_FAMILY,
    fontWeight: 700,
  },
};

 typography: {
    h1: {
      fontSize: '96px',
      ...FONT.light,
    },
    h2: {
      fontSize: '60px',
      ...FONT.regular,
    },
    h3: {
      fontSize: '48px',
      ...FONT.regular,
    },
    h4: {
      fontSize: '34px',
      ...FONT.regular,
    },
    h5: {
      fontSize: '24px',
      ...FONT.regular,
    },
    h6: {
      fontSize: '20px',
      ...FONT.semiBold,
    },
    subtitle1: {
      fontSize: '16px',
      ...FONT.semiBold,
    },
    subtitle2: {
      fontSize: '14px',
      ...FONT.semiBold,
    },
    body1: {
      fontSize: '16px',
      ...FONT.regular,
    },
    body2: {
      fontSize: '14px',
      ...FONT.regular,
    },
    button: {
      fontSize: '15px',
      ...FONT.semiBold,
    },
    caption: {
      fontSize: '12px',
      ...FONT.regular,
    },
    overline: {
      fontSize: '12px',
      ...FONT.regular,
    },
  },