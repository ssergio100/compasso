import { ArrowLeft } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { IconButton } from "../../components/ui/IconButton";

interface AdministrationSubpageHeaderProps {
  clientId: string;
  description: string;
  title: string;
}

export function AdministrationSubpageHeader({ clientId, description, title }: AdministrationSubpageHeaderProps) {
  const navigate = useNavigate();

  return (
    <div className="mb-6 flex items-start gap-3">
      <IconButton label="Voltar para administração" onClick={() => navigate(`/clients/${clientId}/administration`)}>
        <ArrowLeft aria-hidden="true" size={19} />
      </IconButton>
      <div>
        <h1 className="text-lg font-extrabold">{title}</h1>
        <p className="mt-1 text-xs leading-relaxed text-muted">{description}</p>
      </div>
    </div>
  );
}
