import GroupDetail from "@/components/feature/admin/GroupDetail";

export const instant = false;

interface GroupDetailPageProps {
  params: Promise<{ groupId: string }>;
}

export default async function GroupDetailPage({
  params,
}: GroupDetailPageProps) {
  const { groupId } = await params;

  return <GroupDetail groupId={groupId} />;
}
