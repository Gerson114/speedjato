import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "SpeedJato | Administração",
  description: "Agenda, serviços e funcionamento do lava-jato.",
};
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
