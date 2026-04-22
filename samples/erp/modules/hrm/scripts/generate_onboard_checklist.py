def execute(params):
    """Generate onboarding checklist for new employee"""
    employee = params.get("input", {})
    name = employee.get("name", "Unknown")
    department = employee.get("department_id", "general")

    checklist = [
        {"task": "Create email account", "done": False},
        {"task": "Issue ID badge", "done": False},
        {"task": "Setup workstation", "done": False},
        {"task": "Assign mentor", "done": False},
        {"task": f"Department orientation: {department}", "done": False},
        {"task": "HR policy briefing", "done": False},
        {"task": "IT security training", "done": False},
    ]

    return {"employee": name, "checklist": checklist, "total_tasks": len(checklist)}
