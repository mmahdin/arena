import { useNavigate } from "react-router-dom";
import { Compass } from "lucide-react";
import { Page } from "@/components/app/Page";
import { EmptyState } from "@/components/ui/EmptyState";
import { Button } from "@/components/ui/Button";

export function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <Page>
      <div style={{ paddingTop: "18vh" }}>
        <EmptyState
          icon={<Compass size={24} />}
          title="این صفحه پیدا نشد"
          description="نشانی وارد‌شده معتبر نیست یا دسترسی شما به آن بخش محدود شده است."
          action={
            <Button onClick={() => navigate("/")}>بازگشت به خانه</Button>
          }
        />
      </div>
    </Page>
  );
}
