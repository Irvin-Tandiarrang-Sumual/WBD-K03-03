import GroupDetail from "@/components/feature/admin/GroupDetail";

interface GroupDetailPageProps {
  params: Promise<{ groupId: string }>;
}

export default async function GroupDetailPage({
  params,
}: GroupDetailPageProps) {
  const { groupId } = await params;

  return <GroupDetail groupId={groupId} />;
}
