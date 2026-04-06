const OPENINTERN_DEFAULT_AVATAR_OBJECT_KEY =
  "public/system/avatar/openintern-default.jpg";
const OPENINTERN_DEFAULT_AVATAR_ASSET_PATH =
  `/api/backend/v1/assets/${OPENINTERN_DEFAULT_AVATAR_OBJECT_KEY}`;

// Keep site, user, and agent avatar fallbacks aligned to the same default image key.
export const OPENINTERN_DEFAULT_AVATAR_URL =
  OPENINTERN_DEFAULT_AVATAR_ASSET_PATH;
