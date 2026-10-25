import GroupDetail from "@/components/feature/admin/GroupDetail";

export function generateStaticParams() {
  return [
    { groupId: "backend" },
    { groupId: "analitik" },
    { groupId: "qa" },
  ];
}

export default async function GroupDetailPage({ params }) {
  const { groupId } = await params;

  return <GroupDetail groupId={groupId} />;
}
