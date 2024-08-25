Invoke-RestMethod -uri "http://localhost:22222/run40ktestsim" -Method GET
$all = @()
$weapon = @()
$weapon += [pscustomobject]@{
    name="test1"
    attacks = "d6"
    bsws = 3
    strength = "d6+7"
    ap = 2
    damage = "d6"
    }
$targets = @()
$targets += [pscustomobject]@{
    name="test1"
    toughness = 11
    save = 3
    invulnerable_save = 5
    wounds = 2
    fnp = 5
    }

