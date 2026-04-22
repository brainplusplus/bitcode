def execute(params):
    """Calculate remaining leave balance for an employee"""
    employee = params.get("input", {})
    total_annual = 12
    used = params.get("used_days", 0)
    remaining = total_annual - used

    return {
        "employee_id": employee.get("employee_id", ""),
        "total_annual": total_annual,
        "used": used,
        "remaining": max(0, remaining),
        "can_take_leave": remaining > 0,
    }
