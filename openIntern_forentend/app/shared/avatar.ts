const OPENINTERN_DEFAULT_AVATAR_OBJECT_KEY =
  "public/system/avatar/openintern-default.jpg";
const OPENINTERN_DEFAULT_AVATAR_ROOT_RELATIVE_URL =
  `/${OPENINTERN_DEFAULT_AVATAR_OBJECT_KEY}`;

const minioPublicBaseUrl = (
  process.env.NEXT_PUBLIC_MINIO_PUBLIC_BASE_URL ?? ""
).trim();
const normalizedMinioPublicBaseUrl = minioPublicBaseUrl.replace(/\/+$/, "");

// Keep site, user, and agent avatar fallbacks aligned to the same default image key.
export const OPENINTERN_DEFAULT_AVATAR_URL = normalizedMinioPublicBaseUrl
  ? `${normalizedMinioPublicBaseUrl}/${OPENINTERN_DEFAULT_AVATAR_OBJECT_KEY}`
  : OPENINTERN_DEFAULT_AVATAR_ROOT_RELATIVE_URL;
