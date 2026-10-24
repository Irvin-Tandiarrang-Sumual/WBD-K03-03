import { redirect } from "next/navigation";

export const instant = false;

export default async function DatabasePage({ params }) {
  const { title } = await params;
  redirect(`/databases/${encodeURIComponent(title)}/schema`);
}
