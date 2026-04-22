def execute(params):
    """Analyze CRM pipeline - example Python plugin"""
    leads = params.get("leads", [])

    total_revenue = sum(
        l.get("expected_revenue", 0) for l in leads if isinstance(l, dict)
    )
    won_count = sum(
        1 for l in leads if isinstance(l, dict) and l.get("status") == "won"
    )

    return {
        "total_leads": len(leads),
        "total_revenue": total_revenue,
        "won_count": won_count,
        "conversion_rate": (won_count / len(leads) * 100) if leads else 0,
    }
