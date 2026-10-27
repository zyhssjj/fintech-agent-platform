export interface NavItem {
  id: string;
  name: string;
  iconName: string;
  category: "overview" | "fundraising" | "investing" | "risk_control" | "system";
}
