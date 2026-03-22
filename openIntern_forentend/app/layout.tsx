import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";
import "./a2ui-theme.css";
import "../node_modules/@douyinfe/semi-ui-19/dist/css/semi.min.css";
import "./styles/tokens.css";
import "./styles/motion.css";
import "@copilotkit/react-core/v2/styles.css";
import "katex/dist/katex.min.css";
import "highlight.js/styles/github-dark.css";

const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-geist-sans",
});
const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
});

export const metadata: Metadata = {
  title: "openIntern",
  description: "统一管理聊天、Agent、模型、插件与知识库。",
  icons: {
    icon: "/OpenIntern.png",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <head>
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;700;800&family=IBM+Plex+Serif:wght@400;500;600;700&family=Space+Grotesk:wght@400;500;700&family=Google+Sans+Code&display=swap"
        />
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=Google+Symbols:opsz,wght,FILL,GRAD,ROND@20..48,100..700,0..1,-50..200,0..100&display=swap&icon_names=arrow_drop_down,check_circle,close,communication,content_copy,delete,draw,error,info,mobile_layout,pen_size_1,progress_activity,rectangle,send,upload,warning"
        />
      </head>
      <body className={`${geistSans.variable} ${geistMono.variable}`}>
        <a href="#main-content" className="skip-link">
          跳转到主内容
        </a>
        {children}
      </body>
    </html>
  );
}
